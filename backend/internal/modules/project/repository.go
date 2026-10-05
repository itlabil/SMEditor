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

// UpdateStatus sets a project's status and error_message, per the status
// table in docs/flow.md section 3. errorMessage is cleared ("") on a
// successful transition.
func (r *Repository) UpdateStatus(ctx context.Context, id, status, errorMessage string) error {
	const q = `UPDATE projects SET status = ?, error_message = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, status, errorMessage, formatTime(time.Now()), id)
	if err != nil {
		return fmt.Errorf("update project %s status: %w", id, err)
	}
	return nil
}

// UpdateTranscriptLang sets the requested transcript language ("auto",
// "id", "en", "tl", or any other whisper language code), used by
// TranscribeRunner when it next runs.
func (r *Repository) UpdateTranscriptLang(ctx context.Context, id, lang string) error {
	const q = `UPDATE projects SET transcript_lang = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, lang, formatTime(time.Now()), id)
	if err != nil {
		return fmt.Errorf("update transcript language for %s: %w", id, err)
	}
	return nil
}

// DownloadMetadata is what a finished download writes back to the
// project row, per docs/erd.md.
type DownloadMetadata struct {
	VideoTitle  string
	DurationSec float64
	Width       int
	Height      int
	FPS         float64
	VideoCodec  string
	VideoFile   string
	SizeBytes   int64
}

func (r *Repository) SaveDownloadMetadata(ctx context.Context, id string, m DownloadMetadata) error {
	const q = `UPDATE projects SET
		video_title = ?, duration_sec = ?, width = ?, height = ?, fps = ?,
		video_codec = ?, video_file = ?, size_bytes = ?, updated_at = ?
		WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q,
		m.VideoTitle, m.DurationSec, m.Width, m.Height, m.FPS,
		m.VideoCodec, m.VideoFile, m.SizeBytes, formatTime(time.Now()),
		id,
	)
	if err != nil {
		return fmt.Errorf("save download metadata for %s: %w", id, err)
	}
	return nil
}

// UpdateVideoCodecAndSize is used after a convert job re-encodes the
// video in place: the codec and file size change, nothing else.
func (r *Repository) UpdateVideoCodecAndSize(ctx context.Context, id, codec string, sizeBytes int64) error {
	const q = `UPDATE projects SET video_codec = ?, size_bytes = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, codec, sizeBytes, formatTime(time.Now()), id)
	if err != nil {
		return fmt.Errorf("update video codec for %s: %w", id, err)
	}
	return nil
}

// GameExists checks game_modes, a reference table also read directly by
// the prompt module (prompt.Repository.FindGameMode) to assemble
// prompts. project queries it directly here too, rather than calling
// into prompt, since project already depends on prompt (to assemble a
// project's prompt text) and the reverse would be an import cycle.
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
