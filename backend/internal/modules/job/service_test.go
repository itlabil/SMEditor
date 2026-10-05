package job

import (
	"context"
	"errors"
	"testing"
	"time"

	"smeditor/internal/httpx"
)

func newTestService(t *testing.T) (*Service, *Repository, *Worker) {
	t.Helper()
	conn := newTestDB(t)
	seedProject(t, conn, "proj-svc")
	repo := NewRepository(conn)
	hub := NewHub()
	worker := NewWorker(repo, hub, nil)
	return NewService(repo, worker), repo, worker
}

func TestServiceEnqueue_InsertsQueuedJob(t *testing.T) {
	svc, repo, _ := newTestService(t)

	j, err := svc.Enqueue(context.Background(), "proj-svc", "download")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if j.Status != StatusQueued {
		t.Errorf("Status = %q, want %q", j.Status, StatusQueued)
	}

	stored, err := repo.FindByID(context.Background(), j.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if stored.ProjectID != "proj-svc" || stored.Type != "download" {
		t.Errorf("stored job = %+v, want project proj-svc type download", stored)
	}
}

func TestServiceCancel_QueuedJobMarksCanceledDirectly(t *testing.T) {
	svc, repo, _ := newTestService(t)
	j, err := svc.Enqueue(context.Background(), "proj-svc", "download")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	if err := svc.Cancel(context.Background(), j.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	got, err := repo.FindByID(context.Background(), j.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status != StatusCanceled {
		t.Errorf("status = %q, want %q", got.Status, StatusCanceled)
	}
}

func TestServiceCancel_RunningJobWaitsForWorker(t *testing.T) {
	svc, repo, worker := newTestService(t)
	started := make(chan struct{})
	worker.Register(&fakeRunner{jobType: "download", run: func(ctx context.Context, j Job, report ProgressFunc) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}})

	j, err := svc.Enqueue(context.Background(), "proj-svc", "download")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("runner never started")
	}

	if err := svc.Cancel(context.Background(), j.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	got, err := repo.FindByID(context.Background(), j.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status != StatusCanceled {
		t.Errorf("status = %q, want %q", got.Status, StatusCanceled)
	}
}

func TestServiceCancel_AlreadyTerminalIsNoop(t *testing.T) {
	svc, repo, _ := newTestService(t)
	j, err := svc.Enqueue(context.Background(), "proj-svc", "download")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := repo.MarkDone(context.Background(), j.ID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkDone: %v", err)
	}

	if err := svc.Cancel(context.Background(), j.ID); err != nil {
		t.Fatalf("Cancel on already-done job: want nil, got %v", err)
	}

	got, err := repo.FindByID(context.Background(), j.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status != StatusDone {
		t.Errorf("status changed to %q after Cancel, want it to stay %q", got.Status, StatusDone)
	}
}

func TestServiceCancel_NotFound(t *testing.T) {
	svc, _, _ := newTestService(t)

	err := svc.Cancel(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("Cancel with unknown id: want error, got nil")
	}
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Code != "job_not_found" {
		t.Errorf("err = %v, want job_not_found AppError", err)
	}
}

func TestServiceLatestEventForProject_NoJobsReturnsNil(t *testing.T) {
	svc, _, _ := newTestService(t)

	ev, err := svc.LatestEventForProject(context.Background(), "proj-svc")
	if err != nil {
		t.Fatalf("LatestEventForProject: %v", err)
	}
	if ev != nil {
		t.Errorf("ev = %+v, want nil", ev)
	}
}

func TestServiceLatestEventForProject_MapsRunningToProgress(t *testing.T) {
	svc, repo, _ := newTestService(t)
	j, err := svc.Enqueue(context.Background(), "proj-svc", "download")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if err := repo.MarkRunning(context.Background(), j.ID, time.Now().UTC()); err != nil {
		t.Fatalf("MarkRunning: %v", err)
	}
	if err := repo.UpdateProgress(context.Background(), j.ID, 42, "mengunduh..."); err != nil {
		t.Fatalf("UpdateProgress: %v", err)
	}

	ev, err := svc.LatestEventForProject(context.Background(), "proj-svc")
	if err != nil {
		t.Fatalf("LatestEventForProject: %v", err)
	}
	if ev == nil {
		t.Fatal("ev = nil, want an event")
	}
	if ev.Type != "progress" || ev.Progress != 42 || ev.Message != "mengunduh..." {
		t.Errorf("ev = %+v, want progress event with 42%% / mengunduh...", ev)
	}
}
