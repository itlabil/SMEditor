package job

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// ProjectHook lets another module (project) react to job lifecycle events
// without the job module importing it back. A nil hook is a valid no-op,
// used until a concrete implementation exists.
type ProjectHook interface {
	OnJobStarted(ctx context.Context, projectID, jobType string) error
	// outcome is one of StatusDone, StatusFailed, StatusCanceled.
	OnJobFinished(ctx context.Context, projectID, jobType, outcome, message string) error
}

const (
	pollInterval     = 2 * time.Second
	dbWriteInterval  = time.Second
	sseWriteInterval = 250 * time.Millisecond
)

// Worker runs the single background goroutine that drains the jobs queue,
// one job at a time, per .agents/skills/sm-job-worker.
type Worker struct {
	repo    *Repository
	hub     *Hub
	hook    ProjectHook
	runners map[string]Runner
	notify  chan struct{}

	// startMu serializes "pick the next queued job and mark it running"
	// against CancelAllForProject, so a project's own still-queued job can
	// never be picked up in the window between canceling its running job
	// and canceling its queued ones. It is held only for that brief
	// picking step, never for the duration of a Run call.
	startMu sync.Mutex

	mu      sync.Mutex
	current *currentJob
}

type currentJob struct {
	job    Job
	cancel context.CancelFunc
	done   chan struct{}
}

func NewWorker(repo *Repository, hub *Hub, hook ProjectHook) *Worker {
	return &Worker{
		repo:    repo,
		hub:     hub,
		hook:    hook,
		runners: map[string]Runner{},
		notify:  make(chan struct{}, 1),
	}
}

// SetHook assigns the ProjectHook after construction, for wiring where the
// hook implementation itself needs a reference to this worker's Service
// (a construction-order cycle: Worker -> Service -> hook -> Worker). Call
// before RecoverStaleRunning/Start; it is not safe to call once the
// worker is running.
func (w *Worker) SetHook(hook ProjectHook) {
	w.hook = hook
}

// Register adds a Runner for one job type. Call before Start.
func (w *Worker) Register(r Runner) {
	w.runners[r.Type()] = r
}

// NotifyNewJob wakes the worker loop so a freshly queued job is not stuck
// waiting for the next poll tick.
func (w *Worker) NotifyNewJob() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

// RecoverStaleRunning marks every job left in status running (from an app
// restart while it was in flight) as failed, and syncs the owning
// project's status. Call once at startup, before Start.
func (w *Worker) RecoverStaleRunning(ctx context.Context) error {
	stale, err := w.repo.FindRunning(ctx)
	if err != nil {
		return err
	}
	const msg = "app ditutup saat job berjalan"
	for _, j := range stale {
		if err := w.repo.MarkFailed(ctx, j.ID, time.Now().UTC(), msg); err != nil {
			return err
		}
		if w.hook != nil {
			if err := w.hook.OnJobFinished(ctx, j.ProjectID, j.Type, StatusFailed, msg); err != nil {
				return err
			}
		}
	}
	return nil
}

// Start runs the queue-draining loop in a background goroutine until ctx
// is canceled.
func (w *Worker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *Worker) loop(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}
		processed, err := w.processNext(ctx)
		if err != nil {
			log.Printf("job worker: %v", err)
		}
		if processed {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-w.notify:
		case <-ticker.C:
		}
	}
}

// processNext runs the single oldest queued job, if any. It reports
// whether a job was picked up, so the caller can immediately look for the
// next one instead of waiting for the next signal.
//
// Picking the job and marking it running happens under startMu, released
// right after, so a project's queued job can never be dequeued in the gap
// between CancelAllForProject stopping that project's running job and it
// canceling the rest of that project's queue; see the startMu doc comment.
func (w *Worker) processNext(ctx context.Context) (bool, error) {
	w.startMu.Lock()

	j, err := w.repo.FindOldestQueued(ctx)
	if err != nil {
		w.startMu.Unlock()
		return false, err
	}
	if j == nil {
		w.startMu.Unlock()
		return false, nil
	}

	runner, ok := w.runners[j.Type]
	if !ok {
		w.startMu.Unlock()
		msg := "tidak ada runner terdaftar untuk jenis job " + j.Type
		if err := w.finish(ctx, *j, errors.New(msg)); err != nil {
			return true, err
		}
		return true, nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	w.mu.Lock()
	w.current = &currentJob{job: *j, cancel: cancel, done: done}
	w.mu.Unlock()

	if err := w.repo.MarkRunning(ctx, j.ID, time.Now().UTC()); err != nil {
		cancel()
		close(done)
		w.mu.Lock()
		w.current = nil
		w.mu.Unlock()
		w.startMu.Unlock()
		return true, err
	}
	w.startMu.Unlock()

	w.hub.Publish(j.ProjectID, Event{Type: "progress", JobID: j.ID, JobType: j.Type, Progress: 0, Message: ""})
	if w.hook != nil {
		if err := w.hook.OnJobStarted(ctx, j.ProjectID, j.Type); err != nil {
			log.Printf("job worker: OnJobStarted: %v", err)
		}
	}

	report := w.throttledReporter(*j)
	runErr := runner.Run(runCtx, *j, report)
	cancel()

	// finish() persists the terminal status before done closes, so a
	// caller unblocked by <-cur.done (CancelRunning, CancelAllForProject)
	// always sees the final DB state, never a stale "running" row.
	finishErr := w.finish(ctx, *j, runErr)

	w.mu.Lock()
	w.current = nil
	w.mu.Unlock()
	close(done)

	return true, finishErr
}

// finish records the terminal state of a job (err == nil means success)
// and syncs the owning project via the hook.
func (w *Worker) finish(ctx context.Context, j Job, runErr error) error {
	now := time.Now().UTC()
	var outcome, message string

	switch {
	case runErr == nil:
		outcome, message = StatusDone, ""
		if err := w.repo.MarkDone(ctx, j.ID, now); err != nil {
			return err
		}
	case errors.Is(runErr, context.Canceled):
		outcome, message = StatusCanceled, ""
		if err := w.repo.MarkCanceled(ctx, j.ID, now); err != nil {
			return err
		}
	default:
		outcome, message = StatusFailed, runErr.Error()
		if err := w.repo.MarkFailed(ctx, j.ID, now, message); err != nil {
			return err
		}
	}

	w.hub.Publish(j.ProjectID, Event{Type: outcome, JobID: j.ID, JobType: j.Type, Progress: 100, Message: message})
	if w.hook != nil {
		if err := w.hook.OnJobFinished(ctx, j.ProjectID, j.Type, outcome, message); err != nil {
			log.Printf("job worker: OnJobFinished: %v", err)
		}
	}
	return nil
}

func (w *Worker) throttledReporter(j Job) ProgressFunc {
	var mu sync.Mutex
	var lastDB, lastSSE time.Time
	return func(percent float64, message string) {
		mu.Lock()
		defer mu.Unlock()
		now := time.Now()
		if now.Sub(lastDB) >= dbWriteInterval {
			if err := w.repo.UpdateProgress(context.Background(), j.ID, percent, message); err != nil {
				log.Printf("job worker: update progress: %v", err)
			}
			lastDB = now
		}
		if now.Sub(lastSSE) >= sseWriteInterval {
			w.hub.Publish(j.ProjectID, Event{Type: "progress", JobID: j.ID, JobType: j.Type, Progress: percent, Message: message})
			lastSSE = now
		}
	}
}

// CancelRunning cancels id if it is the job currently running, and waits
// until the runner has actually returned. It is a no-op if id is not the
// current job (e.g. it already finished).
func (w *Worker) CancelRunning(ctx context.Context, id string) error {
	w.mu.Lock()
	cur := w.current
	w.mu.Unlock()
	if cur == nil || cur.job.ID != id {
		return nil
	}
	cur.cancel()
	select {
	case <-cur.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CancelAllForProject cancels the running job for projectID, if any, and
// waits for it to stop, then cancels every queued job for the project.
// Used before deleting a project, per .agents/skills/sm-job-worker.
//
// startMu is held for the whole call, so the worker cannot pick up a new
// job (for this project or any other) until the cancellation is complete.
// That briefly pauses the queue, but it is the only way to close the race
// where the project's own queued job gets dequeued in the gap between
// stopping its running job and canceling the rest of its queue.
func (w *Worker) CancelAllForProject(ctx context.Context, projectID string) error {
	w.startMu.Lock()
	defer w.startMu.Unlock()

	w.mu.Lock()
	cur := w.current
	w.mu.Unlock()
	if cur != nil && cur.job.ProjectID == projectID {
		cur.cancel()
		select {
		case <-cur.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return w.repo.CancelAllQueuedForProject(ctx, projectID, time.Now().UTC())
}
