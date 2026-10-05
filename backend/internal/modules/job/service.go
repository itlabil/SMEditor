package job

import (
	"context"
	"errors"
	"time"

	"smeditor/internal/httpx"
	"smeditor/internal/idgen"
)

type Service struct {
	repo   *Repository
	worker *Worker
}

func NewService(repo *Repository, worker *Worker) *Service {
	return &Service{repo: repo, worker: worker}
}

// Enqueue inserts a new queued job for projectID and wakes the worker.
func (s *Service) Enqueue(ctx context.Context, projectID, jobType string) (*Job, error) {
	j := &Job{
		ID:        idgen.New(),
		ProjectID: projectID,
		Type:      jobType,
		Status:    StatusQueued,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, j); err != nil {
		return nil, err
	}
	s.worker.NotifyNewJob()
	return j, nil
}

// Cancel stops job id. A queued job is marked canceled directly; a running
// job is canceled through the worker and this call waits until it has
// actually stopped. A job that already reached a terminal state is a
// no-op.
func (s *Service) Cancel(ctx context.Context, id string) error {
	j, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return httpx.ErrNotFound("job_not_found", "Job tidak ditemukan")
	}
	if err != nil {
		return err
	}

	switch j.Status {
	case StatusQueued:
		return s.repo.CancelQueued(ctx, id, time.Now().UTC())
	case StatusRunning:
		return s.worker.CancelRunning(ctx, id)
	default:
		return nil
	}
}

// CancelAllForProject cancels every job (running or queued) for a
// project, waiting for the running one to actually stop. Used before
// deleting a project.
func (s *Service) CancelAllForProject(ctx context.Context, projectID string) error {
	return s.worker.CancelAllForProject(ctx, projectID)
}

// HasActiveJob reports whether projectID has a job that is queued or
// running, used to reject a retry (e.g. POST /api/projects/:id/download)
// while one is already in flight.
func (s *Service) HasActiveJob(ctx context.Context, projectID string) (bool, error) {
	return s.repo.ExistsActiveForProject(ctx, projectID)
}

// LatestEventForProject reconstructs the event for a project's most
// recent job, so a freshly opened SSE connection can be correct
// immediately, per .agents/skills/sm-job-worker. It returns nil, nil if
// the project has no jobs yet.
func (s *Service) LatestEventForProject(ctx context.Context, projectID string) (*Event, error) {
	j, err := s.repo.FindLatestByProject(ctx, projectID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	evType := j.Status
	if evType == StatusRunning || evType == StatusQueued {
		evType = "progress"
	}
	return &Event{Type: evType, JobID: j.ID, JobType: j.Type, Progress: j.Progress, Message: j.Message}, nil
}
