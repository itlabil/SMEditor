package job

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"smeditor/internal/db"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	return conn
}

func seedProject(t *testing.T, conn *sql.DB, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := conn.Exec(
		`INSERT INTO projects (id, name, game_code, youtube_url, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, "Test Project", "mlbb", "https://youtu.be/x", "baru", now, now,
	)
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

type fakeRunner struct {
	jobType string
	run     func(ctx context.Context, j Job, report ProgressFunc) error
}

func (f *fakeRunner) Type() string { return f.jobType }
func (f *fakeRunner) Run(ctx context.Context, j Job, report ProgressFunc) error {
	return f.run(ctx, j, report)
}

type fakeHook struct {
	mu       sync.Mutex
	started  []string
	finished []struct {
		projectID, jobType, outcome, message string
	}
}

func (f *fakeHook) OnJobStarted(ctx context.Context, projectID, jobType string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, projectID+":"+jobType)
	return nil
}

func (f *fakeHook) OnJobFinished(ctx context.Context, projectID, jobType, outcome, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.finished = append(f.finished, struct{ projectID, jobType, outcome, message string }{projectID, jobType, outcome, message})
	return nil
}

func (f *fakeHook) finishedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.finished)
}

func TestWorkerProcessesQueueOldestFirst(t *testing.T) {
	conn := newTestDB(t)
	seedProject(t, conn, "proj-order")
	repo := NewRepository(conn)
	hub := NewHub()
	worker := NewWorker(repo, hub, nil)

	var mu sync.Mutex
	var order []string
	worker.Register(&fakeRunner{jobType: "noop", run: func(ctx context.Context, j Job, report ProgressFunc) error {
		mu.Lock()
		order = append(order, j.ID)
		mu.Unlock()
		return nil
	}})

	first := &Job{ID: "job-a", ProjectID: "proj-order", Type: "noop", Status: StatusQueued, CreatedAt: time.Now().UTC()}
	second := &Job{ID: "job-b", ProjectID: "proj-order", Type: "noop", Status: StatusQueued, CreatedAt: time.Now().UTC().Add(time.Second)}
	if err := repo.Create(context.Background(), second); err != nil {
		t.Fatalf("create second: %v", err)
	}
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatalf("create first: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)

	waitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(order) == 2
	})

	mu.Lock()
	defer mu.Unlock()
	if order[0] != "job-a" || order[1] != "job-b" {
		t.Errorf("order = %v, want [job-a job-b] (oldest created_at first)", order)
	}

	done, err := repo.FindByID(context.Background(), "job-a")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if done.Status != StatusDone {
		t.Errorf("job-a status = %q, want %q", done.Status, StatusDone)
	}
}

func TestWorkerCancelRunningWaitsForRunnerToStop(t *testing.T) {
	conn := newTestDB(t)
	seedProject(t, conn, "proj-cancel")
	repo := NewRepository(conn)
	hub := NewHub()
	worker := NewWorker(repo, hub, nil)

	started := make(chan struct{})
	worker.Register(&fakeRunner{jobType: "slow", run: func(ctx context.Context, j Job, report ProgressFunc) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}})

	j := &Job{ID: "job-slow", ProjectID: "proj-cancel", Type: "slow", Status: StatusQueued, CreatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), j); err != nil {
		t.Fatalf("create: %v", err)
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	worker.Start(ctx)

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("runner never started")
	}

	if err := worker.CancelRunning(context.Background(), "job-slow"); err != nil {
		t.Fatalf("CancelRunning: %v", err)
	}

	got, err := repo.FindByID(context.Background(), "job-slow")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status != StatusCanceled {
		t.Errorf("status = %q, want %q", got.Status, StatusCanceled)
	}
}

func TestRecoverStaleRunningMarksFailedAndSyncsProject(t *testing.T) {
	conn := newTestDB(t)
	seedProject(t, conn, "proj-stale")
	repo := NewRepository(conn)
	hub := NewHub()
	hook := &fakeHook{}
	worker := NewWorker(repo, hub, hook)

	stuck := &Job{ID: "job-stuck", ProjectID: "proj-stale", Type: "download", Status: StatusRunning, CreatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), stuck); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := worker.RecoverStaleRunning(context.Background()); err != nil {
		t.Fatalf("RecoverStaleRunning: %v", err)
	}

	got, err := repo.FindByID(context.Background(), "job-stuck")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status != StatusFailed {
		t.Errorf("status = %q, want %q", got.Status, StatusFailed)
	}
	if got.Error != "app ditutup saat job berjalan" {
		t.Errorf("error = %q, want the restart message", got.Error)
	}

	if hook.finishedCount() != 1 {
		t.Fatalf("hook.OnJobFinished called %d times, want 1", hook.finishedCount())
	}
	if hook.finished[0].outcome != StatusFailed || hook.finished[0].projectID != "proj-stale" {
		t.Errorf("hook call = %+v, want outcome failed for proj-stale", hook.finished[0])
	}
}

func TestWorkerSkipsJobWithNoRegisteredRunner(t *testing.T) {
	conn := newTestDB(t)
	seedProject(t, conn, "proj-unknown")
	repo := NewRepository(conn)
	hub := NewHub()
	worker := NewWorker(repo, hub, nil)
	// No runner registered for "mystery".

	j := &Job{ID: "job-unknown", ProjectID: "proj-unknown", Type: "mystery", Status: StatusQueued, CreatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), j); err != nil {
		t.Fatalf("create: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)

	waitFor(t, 2*time.Second, func() bool {
		got, err := repo.FindByID(context.Background(), "job-unknown")
		return err == nil && got.Status == StatusFailed
	})
}

func TestCancelAllForProjectStopsRunningAndQueued(t *testing.T) {
	conn := newTestDB(t)
	seedProject(t, conn, "proj-a")
	seedProject(t, conn, "proj-b")
	repo := NewRepository(conn)
	hub := NewHub()
	worker := NewWorker(repo, hub, nil)

	var mu sync.Mutex
	started := map[string]chan struct{}{
		"a-running": make(chan struct{}),
		"b-queued":  make(chan struct{}),
	}
	worker.Register(&fakeRunner{jobType: "slow", run: func(ctx context.Context, j Job, report ProgressFunc) error {
		mu.Lock()
		ch := started[j.ID]
		mu.Unlock()
		close(ch)
		<-ctx.Done()
		return ctx.Err()
	}})

	running := &Job{ID: "a-running", ProjectID: "proj-a", Type: "slow", Status: StatusQueued, CreatedAt: time.Now().UTC()}
	queued := &Job{ID: "a-queued", ProjectID: "proj-a", Type: "slow", Status: StatusQueued, CreatedAt: time.Now().UTC().Add(time.Second)}
	other := &Job{ID: "b-queued", ProjectID: "proj-b", Type: "slow", Status: StatusQueued, CreatedAt: time.Now().UTC()}
	for _, j := range []*Job{running, queued, other} {
		if err := repo.Create(context.Background(), j); err != nil {
			t.Fatalf("create %s: %v", j.ID, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)

	select {
	case <-started["a-running"]:
	case <-time.After(2 * time.Second):
		t.Fatal("runner never started for a-running")
	}

	if err := worker.CancelAllForProject(context.Background(), "proj-a"); err != nil {
		t.Fatalf("CancelAllForProject: %v", err)
	}

	got, err := repo.FindByID(context.Background(), "a-running")
	if err != nil || got.Status != StatusCanceled {
		t.Errorf("a-running status = %+v, %v, want canceled", got, err)
	}
	got, err = repo.FindByID(context.Background(), "a-queued")
	if err != nil || got.Status != StatusCanceled {
		t.Errorf("a-queued status = %+v, %v, want canceled", got, err)
	}
	// b-queued belongs to a different project: CancelAllForProject("proj-a")
	// must never touch it. The worker is free to pick it up right after,
	// so either "queued" (not yet picked up) or "running" (picked up,
	// blocked on ctx like a-running was) proves it was left alone;
	// "canceled" would mean the wrong project's queue was touched.
	got, err = repo.FindByID(context.Background(), "b-queued")
	if err != nil {
		t.Fatalf("FindByID b-queued: %v", err)
	}
	if got.Status == StatusCanceled {
		t.Errorf("b-queued status = %q, want queued or running (untouched by proj-a cancellation)", got.Status)
	}
}

func TestWorkerReportsProgressAndCompletesWithFailure(t *testing.T) {
	conn := newTestDB(t)
	seedProject(t, conn, "proj-fail")
	repo := NewRepository(conn)
	hub := NewHub()
	worker := NewWorker(repo, hub, nil)

	wantErr := errors.New("yt-dlp: unable to download video")
	worker.Register(&fakeRunner{jobType: "download", run: func(ctx context.Context, j Job, report ProgressFunc) error {
		report(50, "mengunduh...")
		return wantErr
	}})

	j := &Job{ID: "job-fail", ProjectID: "proj-fail", Type: "download", Status: StatusQueued, CreatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), j); err != nil {
		t.Fatalf("create: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Start(ctx)

	waitFor(t, 2*time.Second, func() bool {
		got, err := repo.FindByID(context.Background(), "job-fail")
		return err == nil && got.Status == StatusFailed
	})

	got, err := repo.FindByID(context.Background(), "job-fail")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Error != wantErr.Error() {
		t.Errorf("error = %q, want %q", got.Error, wantErr.Error())
	}
}
