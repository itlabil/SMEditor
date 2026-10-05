package job

import (
	"context"
	"time"
)

// Status values, per .agents/skills/sm-job-worker.
const (
	StatusQueued   = "queued"
	StatusRunning  = "running"
	StatusDone     = "done"
	StatusFailed   = "failed"
	StatusCanceled = "canceled"
)

// Job types, per docs/erd.md.
const (
	TypeDownload   = "download"
	TypeConvert    = "convert"
	TypeTranscribe = "transcribe"
	// TypeExport copies a project's finished output to a folder outside
	// data/, per docs/prd.md ("Salin ke folder"). Unlike the other types
	// it never changes the owning project's status (see
	// project.JobSync).
	TypeExport = "export"
)

type Job struct {
	ID        string
	ProjectID string
	Type      string
	Status    string
	Progress  float64
	Message   string
	Error     string
	// Payload is an opaque, job-type-specific string. Only TypeExport
	// uses it (the destination folder to copy into); every other type
	// leaves it empty.
	Payload    string
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
}

// Event is what GET /api/projects/:id/events sends over SSE.
type Event struct {
	Type     string  `json:"type"` // progress, done, failed, canceled
	JobID    string  `json:"job_id"`
	JobType  string  `json:"job_type"`
	Progress float64 `json:"progress"`
	Message  string  `json:"message"`
}

// ProgressFunc is how a Runner reports progress back to the worker. It may
// be called as often as the runner likes; the worker throttles writes.
type ProgressFunc func(percent float64, message string)

// Runner implements one job type (download, convert, transcribe, ...). Run
// must stop promptly when ctx is canceled and return ctx.Err(), per
// .agents/skills/sm-job-worker.
type Runner interface {
	Type() string
	Run(ctx context.Context, j Job, report ProgressFunc) error
}
