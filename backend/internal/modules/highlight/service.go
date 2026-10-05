package highlight

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

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

	saved := SavedHighlight{Game: h.Game, Ringkasan: h.Ringkasan, Segmen: h.Segmen, Video: p.VideoFile, Durasi: p.DurationSec}
	jsonBytes, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("susun highlight.json: %w", err)
	}

	highlightPath, err := s.storage.FilePath(projectID, storage.HighlightFile)
	if err != nil {
		return nil, err
	}
	narasiPath, err := s.storage.FilePath(projectID, storage.NarrationFile)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(highlightPath, jsonBytes, 0o644); err != nil {
		return nil, fmt.Errorf("simpan highlight.json: %w", err)
	}
	if err := os.WriteFile(narasiPath, []byte(buildNarasi(h.Segmen)), 0o644); err != nil {
		return nil, fmt.Errorf("simpan narasi.txt: %w", err)
	}

	if err := s.writer.SetHighlightSaved(ctx, projectID); err != nil {
		return nil, err
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
