package project

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"smeditor/internal/db"
	"smeditor/internal/modules/job"
	"smeditor/internal/storage"
	"smeditor/internal/tools/ffmpeg"
	"smeditor/internal/tools/ytdlp"
)

type fakeYtdlp struct {
	title         string
	titleErr      error
	downloadErr   error
	downloadCalls []ytdlp.DownloadOptions
}

func (f *fakeYtdlp) Title(ctx context.Context, path, url string) (string, error) {
	return f.title, f.titleErr
}

func (f *fakeYtdlp) Download(ctx context.Context, path string, opts ytdlp.DownloadOptions, report ytdlp.ProgressFunc) error {
	f.downloadCalls = append(f.downloadCalls, opts)
	report(50, "separuh jalan")
	return f.downloadErr
}

type fakeFfprobe struct {
	meta ffmpeg.Metadata
	err  error
}

func (f *fakeFfprobe) Probe(ctx context.Context, path, videoPath string) (ffmpeg.Metadata, error) {
	return f.meta, f.err
}

type fakeThumbnailer struct {
	called bool
	err    error
}

func (f *fakeThumbnailer) Thumbnail(ctx context.Context, ffmpegPath, videoPath, outPath string, atSec float64) error {
	f.called = true
	return f.err
}

type fakeSettings struct {
	values map[string]string
}

func (f *fakeSettings) Get(ctx context.Context) (map[string]string, error) {
	return f.values, nil
}

type fakeEnqueuer struct {
	calls []string
	err   error
}

func (f *fakeEnqueuer) Enqueue(ctx context.Context, projectID, jobType string) (*job.Job, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.calls = append(f.calls, jobType)
	return &job.Job{ID: "fake-job", ProjectID: projectID, Type: jobType, Status: job.StatusQueued}, nil
}

func newTestRepoAndStorage(t *testing.T) (*Repository, *storage.Storage) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	return NewRepository(conn), storage.New(t.TempDir())
}

func seedTestProject(t *testing.T, repo *Repository, st *storage.Storage) *Project {
	t.Helper()
	svc := NewService(repo, st, &fakeJobs{}, &fakePromptAssembler{text: "prompt palsu"}, &fakeFolderOpener{}, &fakeSettingsWriter{})
	p, err := svc.Create(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
	return p
}

func TestDownloadRunner_Success_H264EnqueuesTranscribe(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)

	yt := &fakeYtdlp{title: "Judul Video"}
	probe := &fakeFfprobe{meta: ffmpeg.Metadata{
		DurationSec: 100, Width: 1920, Height: 1080, FPS: 24, VideoCodec: "h264", SizeBytes: 12345,
	}}
	thumb := &fakeThumbnailer{}
	jobs := &fakeEnqueuer{}
	settings := &fakeSettings{values: map[string]string{}}

	runner := NewDownloadRunner(repo, st, settings, yt, probe, thumb, jobs)

	var gotPercent float64
	err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeDownload}, func(percent float64, message string) {
		gotPercent = percent
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gotPercent != 50 {
		t.Errorf("reported percent = %v, want 50 (passed through from yt-dlp)", gotPercent)
	}
	if !thumb.called {
		t.Error("Thumbnail was not called")
	}

	got, err := repo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.VideoTitle != "Judul Video" || got.VideoCodec != "h264" || got.DurationSec != 100 {
		t.Errorf("project metadata not saved correctly: %+v", got)
	}

	if len(jobs.calls) != 1 || jobs.calls[0] != job.TypeTranscribe {
		t.Errorf("enqueued = %v, want exactly [transcribe] for an H.264 source", jobs.calls)
	}
}

func TestDownloadRunner_NonH264EnqueuesConvert(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)

	yt := &fakeYtdlp{title: "Judul Video"}
	probe := &fakeFfprobe{meta: ffmpeg.Metadata{VideoCodec: "vp9", DurationSec: 100}}
	jobs := &fakeEnqueuer{}
	runner := NewDownloadRunner(repo, st, &fakeSettings{values: map[string]string{}}, yt, probe, &fakeThumbnailer{}, jobs)

	if err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeDownload}, func(float64, string) {}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(jobs.calls) != 1 || jobs.calls[0] != job.TypeConvert {
		t.Errorf("enqueued = %v, want exactly [convert] for a VP9 source", jobs.calls)
	}
}

func TestDownloadRunner_DownloadError_DoesNotSaveMetadataOrEnqueue(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)

	yt := &fakeYtdlp{downloadErr: errors.New("yt-dlp gagal mengunduh: ERROR: video unavailable")}
	jobs := &fakeEnqueuer{}
	runner := NewDownloadRunner(repo, st, &fakeSettings{values: map[string]string{}}, yt, &fakeFfprobe{}, &fakeThumbnailer{}, jobs)

	err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeDownload}, func(float64, string) {})
	if err == nil {
		t.Fatal("Run with failing download: want error, got nil")
	}

	got, findErr := repo.FindByID(context.Background(), p.ID)
	if findErr != nil {
		t.Fatalf("FindByID: %v", findErr)
	}
	if got.VideoTitle != "" {
		t.Errorf("VideoTitle = %q, want untouched (empty) after a failed download", got.VideoTitle)
	}
	if len(jobs.calls) != 0 {
		t.Errorf("enqueued = %v, want none after a failed download", jobs.calls)
	}
}

func TestDownloadRunner_TitleError_NeverAttemptsDownload(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)

	yt := &fakeYtdlp{titleErr: errors.New("yt-dlp gagal: tidak bisa membaca judul")}
	runner := NewDownloadRunner(repo, st, &fakeSettings{values: map[string]string{}}, yt, &fakeFfprobe{}, &fakeThumbnailer{}, &fakeEnqueuer{})

	err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeDownload}, func(float64, string) {})
	if err == nil {
		t.Fatal("Run with failing title lookup: want error, got nil")
	}
	if len(yt.downloadCalls) != 0 {
		t.Error("Download was called even though Title failed first")
	}
}
