package project

import (
	"context"
	"errors"
	"strings"
	"time"

	"smeditor/internal/httpx"
	"smeditor/internal/idgen"
	"smeditor/internal/modules/job"
	"smeditor/internal/modules/prompt"
	"smeditor/internal/storage"
)

// JobCanceler stops every job (running or queued) for a project before its
// folder is deleted, per .agents/skills/sm-job-worker.
type JobCanceler interface {
	CancelAllForProject(ctx context.Context, projectID string) error
}

// JobEnqueuer queues a new job for a project.
type JobEnqueuer interface {
	Enqueue(ctx context.Context, projectID, jobType string) (*job.Job, error)
}

// JobChecker reports whether a project already has a job in flight, so a
// retry can be rejected instead of racing the existing one.
type JobChecker interface {
	HasActiveJob(ctx context.Context, projectID string) (bool, error)
}

// Jobs is everything the project module needs from the job module,
// implemented by job.Service and injected from internal/app.
type Jobs interface {
	JobCanceler
	JobEnqueuer
	JobChecker
}

// PromptAssembler builds the AI prompt text for a project. Implemented by
// prompt.Service.
type PromptAssembler interface {
	Assemble(ctx context.Context, in prompt.AssembleInput) (string, error)
}

// FolderOpener opens a path in the OS's desktop file manager. Implemented
// by tools.Opener.
type FolderOpener interface {
	OpenFolder(ctx context.Context, path string) error
}

type Service struct {
	repo    *Repository
	storage *storage.Storage
	jobs    Jobs
	prompt  PromptAssembler
	opener  FolderOpener
}

func NewService(repo *Repository, st *storage.Storage, jobs Jobs, promptAssembler PromptAssembler, opener FolderOpener) *Service {
	return &Service{repo: repo, storage: st, jobs: jobs, prompt: promptAssembler, opener: opener}
}

// OpenFolder opens a project's data folder in the OS's file manager, so
// the user can copy the source video and highlight.json into Premiere's
// project folder.
func (s *Service) OpenFolder(ctx context.Context, id string) error {
	if _, err := s.findByID(ctx, id); err != nil {
		return err
	}
	dir, err := s.storage.ProjectDir(id)
	if err != nil {
		return err
	}
	return s.opener.OpenFolder(ctx, dir)
}

// SetHighlightSaved marks a project siap_premiere with has_highlight=1,
// called by the highlight module after it writes highlight.json and
// narasi.txt.
func (s *Service) SetHighlightSaved(ctx context.Context, id string) error {
	if _, err := s.findByID(ctx, id); err != nil {
		return err
	}
	return s.repo.SetHighlightSaved(ctx, id)
}

// ClearHighlight reverts a project to menunggu_highlight with
// has_highlight=0, called by the highlight module after it deletes
// highlight.json and narasi.txt.
func (s *Service) ClearHighlight(ctx context.Context, id string) error {
	if _, err := s.findByID(ctx, id); err != nil {
		return err
	}
	return s.repo.ClearHighlight(ctx, id)
}

// Prompt assembles the AI prompt text for a project, per docs/prd.md.
func (s *Service) Prompt(ctx context.Context, id string) (string, error) {
	p, err := s.findByID(ctx, id)
	if err != nil {
		return "", err
	}
	return s.prompt.Assemble(ctx, prompt.AssembleInput{
		GameCode:      p.GameCode,
		VideoTitle:    p.VideoTitle,
		DurationSec:   p.DurationSec,
		TeamA:         p.TeamA,
		TeamB:         p.TeamB,
		TargetMinutes: p.TargetMinutes,
	})
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Project, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, httpx.ErrUnprocessable("name_required", "Nama project wajib diisi")
	}
	if !IsYoutubeURL(req.YoutubeURL) {
		return nil, httpx.ErrUnprocessable("invalid_url", "URL harus berupa tautan YouTube yang sah")
	}

	gameCode := strings.TrimSpace(req.GameCode)
	exists, err := s.repo.GameExists(ctx, gameCode)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, httpx.ErrUnprocessable("game_not_found", "Kode game tidak dikenal")
	}

	now := time.Now().UTC()
	p := &Project{
		ID:            idgen.New(),
		Name:          name,
		GameCode:      gameCode,
		YoutubeURL:    strings.TrimSpace(req.YoutubeURL),
		TeamA:         strings.TrimSpace(req.TeamA),
		TeamB:         strings.TrimSpace(req.TeamB),
		TargetMinutes: req.TargetMinutes,
		Status:        StatusBaru,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	if _, err := s.storage.EnsureProjectDir(p.ID); err != nil {
		return nil, err
	}
	if _, err := s.jobs.Enqueue(ctx, p.ID, job.TypeDownload); err != nil {
		return nil, err
	}
	return p, nil
}

// RetryDownload queues a new download job for id, e.g. after a failed
// download or to re-fetch after the URL was fixed. It rejects the retry
// if a job is already in flight for this project.
func (s *Service) RetryDownload(ctx context.Context, id string) (*Project, error) {
	if _, err := s.findByID(ctx, id); err != nil {
		return nil, err
	}
	active, err := s.jobs.HasActiveJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, httpx.ErrConflict("job_running", "Masih ada job berjalan untuk project ini")
	}
	if _, err := s.jobs.Enqueue(ctx, id, job.TypeDownload); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// RetryTranscribe queues a new transcribe job for id, optionally
// overriding the requested language ("auto", "id", "en", "tl", or any
// other whisper language code; empty keeps the current one, defaulting
// to "auto"). It rejects the retry if a job is already in flight.
func (s *Service) RetryTranscribe(ctx context.Context, id, lang string) (*Project, error) {
	if _, err := s.findByID(ctx, id); err != nil {
		return nil, err
	}
	active, err := s.jobs.HasActiveJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, httpx.ErrConflict("job_running", "Masih ada job berjalan untuk project ini")
	}

	lang = strings.TrimSpace(lang)
	if lang != "" {
		if err := s.repo.UpdateTranscriptLang(ctx, id, lang); err != nil {
			return nil, err
		}
	}
	if _, err := s.jobs.Enqueue(ctx, id, job.TypeTranscribe); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// TranscriptPath returns the absolute path and suggested download file
// name for a project's transcript, in the requested format ("txt" is the
// default).
func (s *Service) TranscriptPath(ctx context.Context, id, format string) (path, filename string, err error) {
	if _, err := s.findByID(ctx, id); err != nil {
		return "", "", err
	}

	var file string
	switch format {
	case "", "txt":
		file, filename = storage.TranscriptTXTFile, "transcript.txt"
	case "json":
		file, filename = storage.TranscriptJSONFile, "transcript.json"
	default:
		return "", "", httpx.ErrBadRequest("invalid_format", "Format harus txt atau json")
	}

	if !s.storage.Stat(id, file) {
		return "", "", httpx.ErrConflict("transcript_missing", "Transcript belum ada")
	}
	path, err = s.storage.FilePath(id, file)
	if err != nil {
		return "", "", err
	}
	return path, filename, nil
}

func (s *Service) List(ctx context.Context) ([]Project, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].HasThumbnail = s.storage.Stat(list[i].ID, storage.ThumbnailFile)
	}
	return list, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Project, error) {
	p, err := s.findByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.HasThumbnail = s.storage.Stat(p.ID, storage.ThumbnailFile)
	return p, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := s.findByID(ctx, id); err != nil {
		return err
	}
	if err := s.jobs.CancelAllForProject(ctx, id); err != nil {
		return err
	}
	if err := s.storage.DeleteProjectDir(id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// ThumbnailPath returns the absolute path to a project's thumbnail, for the
// handler to serve directly as a file.
func (s *Service) ThumbnailPath(ctx context.Context, id string) (string, error) {
	if _, err := s.findByID(ctx, id); err != nil {
		return "", err
	}
	if !s.storage.Stat(id, storage.ThumbnailFile) {
		return "", httpx.ErrNotFound("thumbnail_not_found", "Thumbnail belum tersedia")
	}
	return s.storage.FilePath(id, storage.ThumbnailFile)
}

// VideoPath returns the absolute path to a project's source video, for
// the handler to serve directly as a file (c.File uses http.ServeContent
// under the hood, which already handles HTTP Range requests, needed for
// the segment preview player to seek without downloading the whole
// file).
func (s *Service) VideoPath(ctx context.Context, id string) (string, error) {
	if _, err := s.findByID(ctx, id); err != nil {
		return "", err
	}
	if !s.storage.Stat(id, storage.SourceVideoFile) {
		return "", httpx.ErrNotFound("video_not_found", "Video belum tersedia")
	}
	return s.storage.FilePath(id, storage.SourceVideoFile)
}

func (s *Service) findByID(ctx context.Context, id string) (*Project, error) {
	if !storage.ValidID(id) {
		return nil, httpx.ErrBadRequest("invalid_id", "ID project tidak valid")
	}
	p, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.ErrNotFound("project_not_found", "Project tidak ditemukan")
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}
