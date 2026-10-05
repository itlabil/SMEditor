package highlight

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"smeditor/internal/httpx"
	"smeditor/internal/modules/project"
	"smeditor/internal/storage"
)

// ProjectReader reads a project's fields needed to validate and save a
// highlight. Implemented by project.Service.
type ProjectReader interface {
	Get(ctx context.Context, id string) (*project.Project, error)
}

// ProjectWriter flips a project's highlight status. Implemented by
// project.Service.
type ProjectWriter interface {
	SetHighlightSaved(ctx context.Context, id string) error
	ClearHighlight(ctx context.Context, id string) error
}

// CategoryLookup returns the valid "kategori" values for a game's genre.
// Implemented by prompt.Service.
type CategoryLookup interface {
	CategoriesForGame(ctx context.Context, gameCode string) ([]string, error)
}

type Service struct {
	storage    *storage.Storage
	projects   ProjectReader
	writer     ProjectWriter
	categories CategoryLookup
}

func NewService(st *storage.Storage, projects ProjectReader, writer ProjectWriter, categories CategoryLookup) *Service {
	return &Service{storage: st, projects: projects, writer: writer, categories: categories}
}

// Save validates rawBody (JSON, optionally wrapped in a code fence)
// against every rule in docs/flow.md section 4.5 and, if it passes,
// writes highlight.json and narasi.txt and marks the project
// siap_premiere. If there is even one error, nothing is written, per
// SM-09's acceptance criteria.
func (s *Service) Save(ctx context.Context, projectID, rawBody string) (*SavedHighlight, error) {
	p, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}

	h, err := Parse(StripCodeFence(rawBody))
	if err != nil {
		return nil, httpx.ErrUnprocessable("highlight_invalid", err.Error())
	}

	saved, err := s.validateAndWrite(ctx, p, h)
	if err != nil {
		return nil, err
	}
	if err := s.writer.SetHighlightSaved(ctx, projectID); err != nil {
		return nil, err
	}
	return saved, nil
}

// UpdateSegment replaces the editable fields of segment nomor (1-based,
// as shown on screen) in the saved highlight, keeping its "alasan". The
// whole highlight is re-checked with the same Validate used by Save; on
// any error nothing is written, per SM-15.
func (s *Service) UpdateSegment(ctx context.Context, projectID string, nomor int, in SegmentInput) (*SavedHighlight, error) {
	p, h, err := s.loadForEdit(ctx, projectID, nomor)
	if err != nil {
		return nil, err
	}
	seg := &h.Segmen[nomor-1]
	seg.Mulai = strings.TrimSpace(in.Mulai)
	seg.Selesai = strings.TrimSpace(in.Selesai)
	seg.Kategori = strings.TrimSpace(in.Kategori)
	seg.Label = strings.TrimSpace(in.Label)
	seg.Narasi = strings.TrimSpace(in.Narasi)
	return s.validateAndWrite(ctx, p, h)
}

// DeleteSegment removes segment nomor (1-based) from the saved highlight
// and rewrites highlight.json and narasi.txt. Removing the last
// remaining segment fails validation ("Tidak ada segmen"); deleting the
// whole highlight has its own endpoint.
func (s *Service) DeleteSegment(ctx context.Context, projectID string, nomor int) (*SavedHighlight, error) {
	p, h, err := s.loadForEdit(ctx, projectID, nomor)
	if err != nil {
		return nil, err
	}
	h.Segmen = append(h.Segmen[:nomor-1], h.Segmen[nomor:]...)
	return s.validateAndWrite(ctx, p, h)
}

// UpdateDraft replaces the saved highlight's draft result (SM-16) and
// rewrites both files after the same validation as Save.
func (s *Service) UpdateDraft(ctx context.Context, projectID string, d Draft) (*SavedHighlight, error) {
	p, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	saved, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	h := saved.toHighlight()
	h.Draft = &d
	return s.validateAndWrite(ctx, p, h)
}

// normalizeDraft trims every name and drops blank hero entries, so a
// trailing comma in the edit form or a stray "" from the AI never ends
// up in highlight.json. Lists are never nil, so they serialize as [].
func normalizeDraft(d *Draft) *Draft {
	if d == nil {
		return nil
	}
	clean := func(names []string) []string {
		out := []string{}
		for _, n := range names {
			if n = strings.TrimSpace(n); n != "" {
				out = append(out, n)
			}
		}
		return out
	}
	team := func(t TeamDraft) TeamDraft {
		return TeamDraft{Nama: strings.TrimSpace(t.Nama), Pick: clean(t.Pick), Ban: clean(t.Ban)}
	}
	return &Draft{TimA: team(d.TimA), TimB: team(d.TimB)}
}

// loadForEdit reads the project and its saved highlight and checks that
// segment nomor exists.
func (s *Service) loadForEdit(ctx context.Context, projectID string, nomor int) (*project.Project, *Highlight, error) {
	p, err := s.projects.Get(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	saved, err := s.Get(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	if nomor < 1 || nomor > len(saved.Segmen) {
		return nil, nil, httpx.ErrNotFound("segment_not_found", fmt.Sprintf("Segmen %d tidak ada", nomor))
	}
	return p, saved.toHighlight(), nil
}

// validateAndWrite runs Validate against the project's genre categories
// and video duration and, only if there are no errors, writes
// highlight.json and narasi.txt.
func (s *Service) validateAndWrite(ctx context.Context, p *project.Project, h *Highlight) (*SavedHighlight, error) {
	h.Draft = normalizeDraft(h.Draft)
	categories, err := s.categories.CategoriesForGame(ctx, p.GameCode)
	if err != nil {
		return nil, err
	}

	if errs := Validate(h, p.DurationSec, categories); len(errs) > 0 {
		return nil, httpx.ErrUnprocessableDetails(
			"highlight_invalid",
			fmt.Sprintf("Ada %d kesalahan pada highlight", len(errs)),
			errs,
		)
	}

	saved := SavedHighlight{Game: h.Game, Ringkasan: h.Ringkasan, Draft: h.Draft, Segmen: h.Segmen, Video: p.VideoFile, Durasi: p.DurationSec}
	jsonBytes, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("susun highlight.json: %w", err)
	}

	highlightPath, err := s.storage.FilePath(p.ID, storage.HighlightFile)
	if err != nil {
		return nil, err
	}
	narasiPath, err := s.storage.FilePath(p.ID, storage.NarrationFile)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(highlightPath, jsonBytes, 0o644); err != nil {
		return nil, fmt.Errorf("simpan highlight.json: %w", err)
	}
	if err := os.WriteFile(narasiPath, []byte(buildNarasi(h.Draft, h.Segmen)), 0o644); err != nil {
		return nil, fmt.Errorf("simpan narasi.txt: %w", err)
	}
	return &saved, nil
}

// Get returns the currently saved highlight, if any.
func (s *Service) Get(ctx context.Context, projectID string) (*SavedHighlight, error) {
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return nil, err
	}
	if !s.storage.Stat(projectID, storage.HighlightFile) {
		return nil, httpx.ErrConflict("highlight_missing", "Highlight belum ada")
	}
	path, err := s.storage.FilePath(projectID, storage.HighlightFile)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("baca highlight.json: %w", err)
	}
	var saved SavedHighlight
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("baca highlight.json: %w", err)
	}
	return &saved, nil
}

// NarasiPath returns the absolute path to narasi.txt, for the handler to
// serve as a file download.
func (s *Service) NarasiPath(ctx context.Context, projectID string) (string, error) {
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return "", err
	}
	if !s.storage.Stat(projectID, storage.NarrationFile) {
		return "", httpx.ErrConflict("highlight_missing", "Highlight belum ada")
	}
	return s.storage.FilePath(projectID, storage.NarrationFile)
}

// Delete removes highlight.json and narasi.txt and reverts the
// project's status to menunggu_highlight, per docs/flow.md section 3.
func (s *Service) Delete(ctx context.Context, projectID string) error {
	if _, err := s.projects.Get(ctx, projectID); err != nil {
		return err
	}

	highlightPath, err := s.storage.FilePath(projectID, storage.HighlightFile)
	if err != nil {
		return err
	}
	narasiPath, err := s.storage.FilePath(projectID, storage.NarrationFile)
	if err != nil {
		return err
	}
	if err := removeIfExists(highlightPath); err != nil {
		return err
	}
	if err := removeIfExists(narasiPath); err != nil {
		return err
	}

	return s.writer.ClearHighlight(ctx, projectID)
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("hapus %s: %w", path, err)
	}
	return nil
}
