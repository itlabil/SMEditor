package project

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("project not found")

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

const projectColumns = `id, name, game_code, youtube_url, video_title, team_a, team_b,
	target_minutes, status, error_message, duration_sec, width, height, fps,
	video_codec, video_file, transcript_lang, has_highlight, size_bytes,
	created_at, updated_at`

func (r *Repository) Create(ctx context.Context, p *Project) error {
	const q = `INSERT INTO projects (` + projectColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, q,
		p.ID, p.Name, p.GameCode, p.YoutubeURL, p.VideoTitle, p.TeamA, p.TeamB,
		p.TargetMinutes, p.Status, p.ErrorMessage, p.DurationSec, p.Width, p.Height, p.FPS,
		p.VideoCodec, p.VideoFile, p.TranscriptLang, boolToInt(p.HasHighlight), p.SizeBytes,
		formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert project %s: %w", p.ID, err)
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Project, error) {
	const q = `SELECT ` + projectColumns + ` FROM projects WHERE id = ?`
	p, err := scanProject(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find project %s: %w", id, err)
	}
	return p, nil
}

func (r *Repository) List(ctx context.Context) ([]Project, error) {
	const q = `SELECT ` + projectColumns + ` FROM projects ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	out := []Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows projects: %w", err)
	}
	return out, nil
}

// Delete removes a project row. jobs for the project are removed by the
// ON DELETE CASCADE on jobs.project_id.
func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected delete project %s: %w", id, err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// GameExists checks game_modes, the reference table of game modes seeded by
// migration. No module owns game_modes yet (it will move behind the prompt
// module in SM-08), so project queries it directly for now.
func (r *Repository) GameExists(ctx context.Context, code string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM game_modes WHERE code = ?`, code).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check game mode %s: %w", code, err)
	}
	return true, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProject(row rowScanner) (*Project, error) {
	var p Project
	var hasHighlight int
	var createdAt, updatedAt string
	err := row.Scan(
		&p.ID, &p.Name, &p.GameCode, &p.YoutubeURL, &p.VideoTitle, &p.TeamA, &p.TeamB,
		&p.TargetMinutes, &p.Status, &p.ErrorMessage, &p.DurationSec, &p.Width, &p.Height, &p.FPS,
		&p.VideoCodec, &p.VideoFile, &p.TranscriptLang, &hasHighlight, &p.SizeBytes,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.HasHighlight = hasHighlight != 0
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &p, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
