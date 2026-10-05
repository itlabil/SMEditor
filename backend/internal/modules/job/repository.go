package job

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("job not found")

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

const jobColumns = `id, project_id, type, status, progress, message, error, payload, created_at, started_at, finished_at`

func (r *Repository) Create(ctx context.Context, j *Job) error {
	const q = `INSERT INTO jobs (` + jobColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, q,
		j.ID, j.ProjectID, j.Type, j.Status, j.Progress, j.Message, j.Error, j.Payload,
		formatTime(j.CreatedAt), formatNullTime(j.StartedAt), formatNullTime(j.FinishedAt),
	)
	if err != nil {
		return fmt.Errorf("insert job %s: %w", j.ID, err)
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*Job, error) {
	const q = `SELECT ` + jobColumns + ` FROM jobs WHERE id = ?`
	j, err := scanJob(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find job %s: %w", id, err)
	}
	return j, nil
}

// FindOldestQueued returns the longest-waiting queued job, or nil if the
// queue is empty.
func (r *Repository) FindOldestQueued(ctx context.Context) (*Job, error) {
	const q = `SELECT ` + jobColumns + ` FROM jobs WHERE status = ? ORDER BY created_at ASC LIMIT 1`
	j, err := scanJob(r.db.QueryRowContext(ctx, q, StatusQueued))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find oldest queued job: %w", err)
	}
	return j, nil
}

// FindRunning returns every job left in status running, used to recover
// from an app restart while a job was in flight.
func (r *Repository) FindRunning(ctx context.Context) ([]Job, error) {
	const q = `SELECT ` + jobColumns + ` FROM jobs WHERE status = ?`
	rows, err := r.db.QueryContext(ctx, q, StatusRunning)
	if err != nil {
		return nil, fmt.Errorf("query running jobs: %w", err)
	}
	defer rows.Close()

	out := []Job{}
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan running job: %w", err)
		}
		out = append(out, *j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows running jobs: %w", err)
	}
	return out, nil
}

// ExistsActiveForProject reports whether projectID has a job that is
// queued or running, used to reject a retry while one is already in
// flight (the job_running error).
func (r *Repository) ExistsActiveForProject(ctx context.Context, projectID string) (bool, error) {
	const q = `SELECT 1 FROM jobs WHERE project_id = ? AND status IN (?, ?) LIMIT 1`
	var exists int
	err := r.db.QueryRowContext(ctx, q, projectID, StatusQueued, StatusRunning).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check active job for project %s: %w", projectID, err)
	}
	return true, nil
}

func (r *Repository) FindLatestByProject(ctx context.Context, projectID string) (*Job, error) {
	const q = `SELECT ` + jobColumns + ` FROM jobs WHERE project_id = ? ORDER BY created_at DESC LIMIT 1`
	j, err := scanJob(r.db.QueryRowContext(ctx, q, projectID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find latest job for project %s: %w", projectID, err)
	}
	return j, nil
}

func (r *Repository) MarkRunning(ctx context.Context, id string, startedAt time.Time) error {
	const q = `UPDATE jobs SET status = ?, started_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, StatusRunning, formatTime(startedAt), id)
	if err != nil {
		return fmt.Errorf("mark job %s running: %w", id, err)
	}
	return nil
}

func (r *Repository) MarkDone(ctx context.Context, id string, finishedAt time.Time) error {
	const q = `UPDATE jobs SET status = ?, progress = 100, finished_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, StatusDone, formatTime(finishedAt), id)
	if err != nil {
		return fmt.Errorf("mark job %s done: %w", id, err)
	}
	return nil
}

func (r *Repository) MarkFailed(ctx context.Context, id string, finishedAt time.Time, message string) error {
	const q = `UPDATE jobs SET status = ?, error = ?, finished_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, StatusFailed, message, formatTime(finishedAt), id)
	if err != nil {
		return fmt.Errorf("mark job %s failed: %w", id, err)
	}
	return nil
}

func (r *Repository) MarkCanceled(ctx context.Context, id string, finishedAt time.Time) error {
	const q = `UPDATE jobs SET status = ?, finished_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, StatusCanceled, formatTime(finishedAt), id)
	if err != nil {
		return fmt.Errorf("mark job %s canceled: %w", id, err)
	}
	return nil
}

// CancelQueued cancels id only if it is still queued; it is a no-op if the
// job already started or finished, since the caller may race with the
// worker picking it up.
func (r *Repository) CancelQueued(ctx context.Context, id string, finishedAt time.Time) error {
	const q = `UPDATE jobs SET status = ?, finished_at = ? WHERE id = ? AND status = ?`
	_, err := r.db.ExecContext(ctx, q, StatusCanceled, formatTime(finishedAt), id, StatusQueued)
	if err != nil {
		return fmt.Errorf("cancel queued job %s: %w", id, err)
	}
	return nil
}

func (r *Repository) CancelAllQueuedForProject(ctx context.Context, projectID string, finishedAt time.Time) error {
	const q = `UPDATE jobs SET status = ?, finished_at = ? WHERE project_id = ? AND status = ?`
	_, err := r.db.ExecContext(ctx, q, StatusCanceled, formatTime(finishedAt), projectID, StatusQueued)
	if err != nil {
		return fmt.Errorf("cancel queued jobs for project %s: %w", projectID, err)
	}
	return nil
}

func (r *Repository) UpdateProgress(ctx context.Context, id string, percent float64, message string) error {
	const q = `UPDATE jobs SET progress = ?, message = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, q, percent, message, id)
	if err != nil {
		return fmt.Errorf("update progress for job %s: %w", id, err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(row rowScanner) (*Job, error) {
	var j Job
	var createdAt string
	var startedAt, finishedAt sql.NullString
	err := row.Scan(
		&j.ID, &j.ProjectID, &j.Type, &j.Status, &j.Progress, &j.Message, &j.Error, &j.Payload,
		&createdAt, &startedAt, &finishedAt,
	)
	if err != nil {
		return nil, err
	}
	j.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	j.StartedAt = parseNullTime(startedAt)
	j.FinishedAt = parseNullTime(finishedAt)
	return &j, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func formatNullTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(*t), Valid: true}
}

func parseNullTime(s sql.NullString) *time.Time {
	if !s.Valid {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s.String)
	if err != nil {
		return nil
	}
	return &t
}
