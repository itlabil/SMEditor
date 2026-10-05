package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"smeditor/internal/db"
	"smeditor/internal/httpx"
	"smeditor/internal/modules/job"
	"smeditor/internal/storage"
)

type fakeJobs struct {
	calledForProject string // last CancelAllForProject target
	enqueued         []string
	active           bool
	enqueueErr       error
}

func (f *fakeJobs) CancelAllForProject(ctx context.Context, projectID string) error {
	f.calledForProject = projectID
	return nil
}

func (f *fakeJobs) Enqueue(ctx context.Context, projectID, jobType string) (*job.Job, error) {
	if f.enqueueErr != nil {
		return nil, f.enqueueErr
	}
	f.enqueued = append(f.enqueued, jobType)
	return &job.Job{ID: "fake-job", ProjectID: projectID, Type: jobType, Status: job.StatusQueued}, nil
}

func (f *fakeJobs) HasActiveJob(ctx context.Context, projectID string) (bool, error) {
	return f.active, nil
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc, _ := newTestServiceWithJobs(t)
	return svc
}

func newTestServiceWithJobs(t *testing.T) (*Service, *fakeJobs) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}

	repo := NewRepository(conn)
	st := storage.New(t.TempDir())
	jobs := &fakeJobs{}
	return NewService(repo, st, jobs), jobs
}

func validRequest() CreateRequest {
	return CreateRequest{
		Name:       "MPL Game 3",
		GameCode:   "mlbb",
		YoutubeURL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	}
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error %v is not an *httpx.AppError", err)
	}
	return appErr.Code
}

func TestServiceCreate_Success(t *testing.T) {
	svc, jobs := newTestServiceWithJobs(t)

	p, err := svc.Create(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Status != StatusBaru {
		t.Errorf("Status = %q, want %q", p.Status, StatusBaru)
	}
	if !storage.ValidID(p.ID) {
		t.Errorf("ID %q is not a valid ULID", p.ID)
	}

	dir, err := svc.storage.ProjectDir(p.ID)
	if err != nil {
		t.Fatalf("ProjectDir: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("project dir %s was not created", dir)
	}

	if len(jobs.enqueued) != 1 || jobs.enqueued[0] != job.TypeDownload {
		t.Errorf("enqueued jobs = %v, want exactly one %q", jobs.enqueued, job.TypeDownload)
	}
}

func TestServiceCreate_NameRequired(t *testing.T) {
	svc := newTestService(t)
	req := validRequest()
	req.Name = "   "

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("Create with empty name: want error, got nil")
	}
	if code := appErrCode(t, err); code != "name_required" {
		t.Errorf("code = %q, want name_required", code)
	}
}

func TestServiceCreate_InvalidURL(t *testing.T) {
	svc := newTestService(t)
	req := validRequest()
	req.YoutubeURL = "https://vimeo.com/12345"

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("Create with non-YouTube URL: want error, got nil")
	}
	if code := appErrCode(t, err); code != "invalid_url" {
		t.Errorf("code = %q, want invalid_url", code)
	}
}

func TestServiceCreate_GameNotFound(t *testing.T) {
	svc := newTestService(t)
	req := validRequest()
	req.GameCode = "not-a-real-game"

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("Create with unknown game code: want error, got nil")
	}
	if code := appErrCode(t, err); code != "game_not_found" {
		t.Errorf("code = %q, want game_not_found", code)
	}
}

func TestServiceGet_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Get(context.Background(), "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err == nil {
		t.Fatal("Get with unknown id: want error, got nil")
	}
	if code := appErrCode(t, err); code != "project_not_found" {
		t.Errorf("code = %q, want project_not_found", code)
	}
}

func TestServiceGet_InvalidID(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Get(context.Background(), "../..")
	if err == nil {
		t.Fatal("Get with path-traversal id: want error, got nil")
	}
	if code := appErrCode(t, err); code != "invalid_id" {
		t.Errorf("code = %q, want invalid_id", code)
	}
}

func TestServiceList_ReturnsCreatedProjects(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, validRequest()); err != nil {
		t.Fatalf("Create: %v", err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List() len = %d, want 1", len(list))
	}
	if list[0].Name != "MPL Game 3" {
		t.Errorf("List()[0].Name = %q, want %q", list[0].Name, "MPL Game 3")
	}
}

func TestServiceDelete_RemovesRowAndFolder(t *testing.T) {
	svc, jobs := newTestServiceWithJobs(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	dir, err := svc.storage.ProjectDir(p.ID)
	if err != nil {
		t.Fatalf("ProjectDir: %v", err)
	}

	if err := svc.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("project dir %s still exists after delete", dir)
	}
	if _, err := svc.Get(ctx, p.ID); err == nil {
		t.Fatal("Get after Delete: want error, got nil")
	}
	if jobs.calledForProject != p.ID {
		t.Errorf("Delete did not cancel jobs for project before removing it: got %q, want %q", jobs.calledForProject, p.ID)
	}
}

func TestServiceDelete_NotFound(t *testing.T) {
	svc := newTestService(t)

	err := svc.Delete(context.Background(), "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err == nil {
		t.Fatal("Delete with unknown id: want error, got nil")
	}
	if code := appErrCode(t, err); code != "project_not_found" {
		t.Errorf("code = %q, want project_not_found", code)
	}
}

func TestServiceDelete_RejectsPathTraversalID(t *testing.T) {
	svc := newTestService(t)

	err := svc.Delete(context.Background(), "../..")
	if err == nil {
		t.Fatal("Delete with path-traversal id: want error, got nil")
	}
	if code := appErrCode(t, err); code != "invalid_id" {
		t.Errorf("code = %q, want invalid_id", code)
	}
}

func TestServiceRetryDownload_EnqueuesWhenIdle(t *testing.T) {
	svc, jobs := newTestServiceWithJobs(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	jobs.enqueued = nil // reset what Create itself enqueued

	got, err := svc.RetryDownload(ctx, p.ID)
	if err != nil {
		t.Fatalf("RetryDownload: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("RetryDownload returned project %q, want %q", got.ID, p.ID)
	}
	if len(jobs.enqueued) != 1 || jobs.enqueued[0] != job.TypeDownload {
		t.Errorf("enqueued jobs = %v, want exactly one %q", jobs.enqueued, job.TypeDownload)
	}
}

func TestServiceRetryDownload_RejectsWhenJobActive(t *testing.T) {
	svc, jobs := newTestServiceWithJobs(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	jobs.enqueued = nil
	jobs.active = true

	_, err = svc.RetryDownload(ctx, p.ID)
	if err == nil {
		t.Fatal("RetryDownload while a job is active: want error, got nil")
	}
	if code := appErrCode(t, err); code != "job_running" {
		t.Errorf("code = %q, want job_running", code)
	}
	if len(jobs.enqueued) != 0 {
		t.Errorf("enqueued jobs = %v, want none", jobs.enqueued)
	}
}

func TestServiceRetryDownload_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.RetryDownload(context.Background(), "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err == nil {
		t.Fatal("RetryDownload with unknown id: want error, got nil")
	}
	if code := appErrCode(t, err); code != "project_not_found" {
		t.Errorf("code = %q, want project_not_found", code)
	}
}

func TestServiceRetryTranscribe_EnqueuesAndUpdatesLanguage(t *testing.T) {
	svc, jobs := newTestServiceWithJobs(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	jobs.enqueued = nil

	got, err := svc.RetryTranscribe(ctx, p.ID, "id")
	if err != nil {
		t.Fatalf("RetryTranscribe: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("RetryTranscribe returned project %q, want %q", got.ID, p.ID)
	}
	if len(jobs.enqueued) != 1 || jobs.enqueued[0] != job.TypeTranscribe {
		t.Errorf("enqueued jobs = %v, want exactly one %q", jobs.enqueued, job.TypeTranscribe)
	}

	stored, err := svc.repo.FindByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if stored.TranscriptLang != "id" {
		t.Errorf("TranscriptLang = %q, want id", stored.TranscriptLang)
	}
}

func TestServiceRetryTranscribe_EmptyLangKeepsExisting(t *testing.T) {
	svc, jobs := newTestServiceWithJobs(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.repo.UpdateTranscriptLang(ctx, p.ID, "en"); err != nil {
		t.Fatalf("UpdateTranscriptLang: %v", err)
	}
	jobs.enqueued = nil

	if _, err := svc.RetryTranscribe(ctx, p.ID, ""); err != nil {
		t.Fatalf("RetryTranscribe: %v", err)
	}

	stored, err := svc.repo.FindByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if stored.TranscriptLang != "en" {
		t.Errorf("TranscriptLang = %q, want it to stay en when retry sends no override", stored.TranscriptLang)
	}
}

func TestServiceRetryTranscribe_RejectsWhenJobActive(t *testing.T) {
	svc, jobs := newTestServiceWithJobs(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	jobs.active = true

	_, err = svc.RetryTranscribe(ctx, p.ID, "")
	if err == nil {
		t.Fatal("RetryTranscribe while a job is active: want error, got nil")
	}
	if code := appErrCode(t, err); code != "job_running" {
		t.Errorf("code = %q, want job_running", code)
	}
}

func TestServiceTranscriptPath_MissingReturnsConflict(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, _, err = svc.TranscriptPath(ctx, p.ID, "txt")
	if err == nil {
		t.Fatal("TranscriptPath before transcript exists: want error, got nil")
	}
	if code := appErrCode(t, err); code != "transcript_missing" {
		t.Errorf("code = %q, want transcript_missing", code)
	}
}

func TestServiceTranscriptPath_InvalidFormat(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, _, err = svc.TranscriptPath(ctx, p.ID, "pdf")
	if err == nil {
		t.Fatal("TranscriptPath with invalid format: want error, got nil")
	}
	if code := appErrCode(t, err); code != "invalid_format" {
		t.Errorf("code = %q, want invalid_format", code)
	}
}

func TestServiceTranscriptPath_ReturnsPathWhenPresent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	path, err := svc.storage.FilePath(p.ID, storage.TranscriptTXTFile)
	if err != nil {
		t.Fatalf("FilePath: %v", err)
	}
	if err := os.WriteFile(path, []byte("[00:00:00] halo\n"), 0o644); err != nil {
		t.Fatalf("seed transcript.txt: %v", err)
	}

	got, filename, err := svc.TranscriptPath(ctx, p.ID, "txt")
	if err != nil {
		t.Fatalf("TranscriptPath: %v", err)
	}
	if got != path || filename != "transcript.txt" {
		t.Errorf("TranscriptPath = (%q, %q), want (%q, %q)", got, filename, path, "transcript.txt")
	}
}
