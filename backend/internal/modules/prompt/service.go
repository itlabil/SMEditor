package prompt

import (
	"context"
	"errors"
	"strings"

	"smeditor/internal/httpx"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) ListGameModes(ctx context.Context) ([]GameMode, error) {
	return s.repo.ListGameModes(ctx)
}

func (s *Service) GetBlock(ctx context.Context, code string) (*Block, error) {
	b, err := s.repo.FindBlock(ctx, code)
	if errors.Is(err, ErrBlockNotFound) {
		return nil, httpx.ErrNotFound("prompt_block_not_found", "Blok prompt tidak ditemukan")
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) UpdateBlock(ctx context.Context, code string, req UpdateBlockRequest) (*Block, error) {
	if _, ok := defaultBlocks[code]; !ok {
		return nil, httpx.ErrNotFound("prompt_block_not_found", "Blok prompt tidak ditemukan")
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, httpx.ErrUnprocessable("prompt_body_required", "Isi blok prompt wajib diisi")
	}
	categories := req.Categories
	if categories == nil {
		categories = []string{}
	}
	if err := s.repo.UpdateBlock(ctx, code, body, categories); err != nil {
		return nil, err
	}
	return s.GetBlock(ctx, code)
}

// ResetBlock rewrites a block back to its seed content, per docs/prd.md
// ("Tombol 'kembalikan ke bawaan' menulis ulang dari seed").
func (s *Service) ResetBlock(ctx context.Context, code string) (*Block, error) {
	def, ok := defaultBlocks[code]
	if !ok {
		return nil, httpx.ErrNotFound("prompt_block_not_found", "Blok prompt tidak ditemukan")
	}
	if err := s.repo.ResetBlock(ctx, code, def.Body, def.Categories); err != nil {
		return nil, err
	}
	return s.GetBlock(ctx, code)
}

// Assemble builds the full prompt text for one project, per docs/prd.md
// ("Kerangka bersama" + blok tugas + istilah game).
func (s *Service) Assemble(ctx context.Context, in AssembleInput) (string, error) {
	gm, err := s.repo.FindGameMode(ctx, in.GameCode)
	if errors.Is(err, ErrGameNotFound) {
		return "", httpx.ErrUnprocessable("game_not_found", "Kode game tidak dikenal")
	}
	if err != nil {
		return "", err
	}

	frame, err := s.repo.FindBlock(ctx, "frame")
	if err != nil {
		return "", err
	}
	block, err := s.repo.FindBlock(ctx, gm.BlockCode)
	if err != nil {
		return "", err
	}

	return assemble(*frame, *block, *gm, in), nil
}

// CategoriesForGame returns the valid highlight categories for a game's
// genre block, used by the highlight module to validate a segment's
// "kategori" field, per docs/flow.md section 4.5.
func (s *Service) CategoriesForGame(ctx context.Context, gameCode string) ([]string, error) {
	gm, err := s.repo.FindGameMode(ctx, gameCode)
	if errors.Is(err, ErrGameNotFound) {
		return nil, httpx.ErrUnprocessable("game_not_found", "Kode game tidak dikenal")
	}
	if err != nil {
		return nil, err
	}
	block, err := s.repo.FindBlock(ctx, gm.BlockCode)
	if err != nil {
		return nil, err
	}
	return block.Categories, nil
}
