package prompt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrGameNotFound  = errors.New("game mode not found")
	ErrBlockNotFound = errors.New("prompt block not found")
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListGameModes(ctx context.Context) ([]GameMode, error) {
	const q = `SELECT code, name, genre, block_code, terms_json, sort_order FROM game_modes ORDER BY sort_order`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query game modes: %w", err)
	}
	defer rows.Close()

	out := []GameMode{}
	for rows.Next() {
		gm, err := scanGameMode(rows)
		if err != nil {
			return nil, fmt.Errorf("scan game mode: %w", err)
		}
		out = append(out, *gm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows game modes: %w", err)
	}
	return out, nil
}

func (r *Repository) FindGameMode(ctx context.Context, code string) (*GameMode, error) {
	const q = `SELECT code, name, genre, block_code, terms_json, sort_order FROM game_modes WHERE code = ?`
	gm, err := scanGameMode(r.db.QueryRowContext(ctx, q, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGameNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find game mode %s: %w", code, err)
	}
	return gm, nil
}

func (r *Repository) FindBlock(ctx context.Context, code string) (*Block, error) {
	const q = `SELECT code, title, body, categories_json, is_custom, updated_at FROM prompt_blocks WHERE code = ?`
	b, err := scanBlock(r.db.QueryRowContext(ctx, q, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBlockNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find prompt block %s: %w", code, err)
	}
	return b, nil
}

func (r *Repository) UpdateBlock(ctx context.Context, code, body string, categories []string) error {
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return fmt.Errorf("marshal categories: %w", err)
	}
	const q = `UPDATE prompt_blocks SET body = ?, categories_json = ?, is_custom = 1, updated_at = ? WHERE code = ?`
	res, err := r.db.ExecContext(ctx, q, body, string(categoriesJSON), formatTime(time.Now()), code)
	if err != nil {
		return fmt.Errorf("update prompt block %s: %w", code, err)
	}
	return checkRowsAffected(res, code)
}

func (r *Repository) ResetBlock(ctx context.Context, code, body string, categories []string) error {
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return fmt.Errorf("marshal categories: %w", err)
	}
	const q = `UPDATE prompt_blocks SET body = ?, categories_json = ?, is_custom = 0, updated_at = ? WHERE code = ?`
	res, err := r.db.ExecContext(ctx, q, body, string(categoriesJSON), formatTime(time.Now()), code)
	if err != nil {
		return fmt.Errorf("reset prompt block %s: %w", code, err)
	}
	return checkRowsAffected(res, code)
}

func checkRowsAffected(res sql.Result, code string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for %s: %w", code, err)
	}
	if n == 0 {
		return ErrBlockNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanGameMode(row rowScanner) (*GameMode, error) {
	var gm GameMode
	var termsJSON string
	if err := row.Scan(&gm.Code, &gm.Name, &gm.Genre, &gm.BlockCode, &termsJSON, &gm.SortOrder); err != nil {
		return nil, err
	}
	gm.Terms = map[string]string{}
	if err := json.Unmarshal([]byte(termsJSON), &gm.Terms); err != nil {
		return nil, fmt.Errorf("parse terms_json for %s: %w", gm.Code, err)
	}
	return &gm, nil
}

func scanBlock(row rowScanner) (*Block, error) {
	var b Block
	var categoriesJSON, updatedAt string
	var isCustom int
	if err := row.Scan(&b.Code, &b.Title, &b.Body, &categoriesJSON, &isCustom, &updatedAt); err != nil {
		return nil, err
	}
	b.Categories = []string{}
	if err := json.Unmarshal([]byte(categoriesJSON), &b.Categories); err != nil {
		return nil, fmt.Errorf("parse categories_json for %s: %w", b.Code, err)
	}
	b.IsCustom = isCustom != 0
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &b, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
