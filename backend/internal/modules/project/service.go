package project

import (
	"context"
	"errors"
	"strings"
	"time"

	"smeditor/internal/httpx"
	"smeditor/internal/idgen"
	"smeditor/internal/storage"
)

type Service struct {
	repo    *Repository
	storage *storage.Storage
}

func NewService(repo *Repository, st *storage.Storage) *Service {
	return &Service{repo: repo, storage: st}
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
	return p, nil
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
