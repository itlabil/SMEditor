package project

import (
	"context"

	"smeditor/internal/modules/job"
)

// JobSync implements job.ProjectHook: it keeps a project's status in sync
// with its jobs' lifecycle, per the status table in docs/flow.md section 3
// and .agents/skills/sm-job-worker.
//
// It only ever reacts generically to a job's type and outcome. The one
// decision that needs fresher information (whether a finished download
// needs a convert pass before transcribing) is made by DownloadRunner
// itself, which has the just-probed codec in hand; this hook does nothing
// for a "download" job reaching StatusDone.
type JobSync struct {
	repo *Repository
	jobs JobEnqueuer
}

func NewJobSync(repo *Repository, jobs JobEnqueuer) *JobSync {
	return &JobSync{repo: repo, jobs: jobs}
}

func (s *JobSync) OnJobStarted(ctx context.Context, projectID, jobType string) error {
	status, ok := startStatus(jobType)
	if !ok {
		return nil
	}
	return s.repo.UpdateStatus(ctx, projectID, status, "")
}

func (s *JobSync) OnJobFinished(ctx context.Context, projectID, jobType, outcome, message string) error {
	switch outcome {
	case job.StatusFailed:
		status, ok := failStatus(jobType)
		if !ok {
			return nil
		}
		return s.repo.UpdateStatus(ctx, projectID, status, message)
	case job.StatusDone:
		return s.onDone(ctx, projectID, jobType)
	default:
		// canceled: leave the project's status as it was; the user can retry.
		return nil
	}
}

func (s *JobSync) onDone(ctx context.Context, projectID, jobType string) error {
	switch jobType {
	case job.TypeConvert:
		_, err := s.jobs.Enqueue(ctx, projectID, job.TypeTranscribe)
		return err
	case job.TypeTranscribe:
		return s.repo.UpdateStatus(ctx, projectID, StatusMenungguHighlight, "")
	}
	return nil
}

func startStatus(jobType string) (string, bool) {
	switch jobType {
	case job.TypeDownload, job.TypeConvert:
		return StatusMengunduh, true
	case job.TypeTranscribe:
		return StatusTranscript, true
	}
	return "", false
}

func failStatus(jobType string) (string, bool) {
	switch jobType {
	case job.TypeDownload, job.TypeConvert:
		return StatusGagalUnduh, true
	case job.TypeTranscribe:
		return StatusGagalTranscript, true
	}
	return "", false
}
