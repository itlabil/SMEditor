package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"smeditor/internal/modules/job"
	"smeditor/internal/storage"
)

func seedExportFiles(t *testing.T, st *storage.Storage, projectID string) {
	t.Helper()
	contents := map[string]string{
		storage.SourceVideoFile: "fake video bytes",
		storage.HighlightFile:   `{"video":"source.mp4","durasi":12}`,
		storage.NarrationFile:   "narasi baris 1\nnarasi baris 2\n",
	}
	for name, content := range contents {
		path, err := st.FilePath(projectID, name)
		if err != nil {
			t.Fatalf("FilePath(%s): %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
}

func TestExportRunner_Success_CopiesAllThreeFiles(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)
	seedExportFiles(t, st, p.ID)

	targetDir := filepath.Join(t.TempDir(), "MyProject")
	runner := NewExportRunner(repo, st)

	var lastPercent float64
	err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeExport, Payload: targetDir}, func(percent float64, message string) {
		lastPercent = percent
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if lastPercent != 100 {
		t.Errorf("final reported percent = %v, want 100", lastPercent)
	}

	for _, name := range exportFiles {
		srcPath, err := st.FilePath(p.ID, name)
		if err != nil {
			t.Fatalf("FilePath(%s): %v", name, err)
		}
		want, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("read source %s: %v", name, err)
		}
		got, err := os.ReadFile(filepath.Join(targetDir, name))
		if err != nil {
			t.Fatalf("read copied %s: %v", name, err)
		}
		if string(got) != string(want) {
			t.Errorf("copied %s content = %q, want %q", name, got, want)
		}
		if _, err := os.Stat(filepath.Join(targetDir, name+exportTempSuffix)); !os.IsNotExist(err) {
			t.Errorf("leftover temp file for %s after success", name)
		}
	}
}

func TestExportRunner_HighlightVideoFieldMatchesCopiedVideoName(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)
	seedExportFiles(t, st, p.ID)

	targetDir := filepath.Join(t.TempDir(), "MyProject")
	runner := NewExportRunner(repo, st)
	if err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeExport, Payload: targetDir}, func(float64, string) {}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	highlightBytes, err := os.ReadFile(filepath.Join(targetDir, storage.HighlightFile))
	if err != nil {
		t.Fatalf("read copied highlight.json: %v", err)
	}
	if !strings.Contains(string(highlightBytes), `"video":"`+storage.SourceVideoFile+`"`) {
		t.Errorf("highlight.json video field does not reference %q: %s", storage.SourceVideoFile, highlightBytes)
	}
	if _, err := os.Stat(filepath.Join(targetDir, storage.SourceVideoFile)); err != nil {
		t.Errorf("copied video not found under the name highlight.json references: %v", err)
	}
}

func TestExportRunner_RemovesLeftoverTempFileFromPreviousAttempt(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)
	seedExportFiles(t, st, p.ID)

	targetDir := filepath.Join(t.TempDir(), "MyProject")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	stale := filepath.Join(targetDir, storage.SourceVideoFile+exportTempSuffix)
	if err := os.WriteFile(stale, []byte("half-written from a crashed attempt"), 0o644); err != nil {
		t.Fatalf("seed stale temp file: %v", err)
	}

	runner := NewExportRunner(repo, st)
	if err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeExport, Payload: targetDir}, func(float64, string) {}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale temp file from a previous attempt was not removed")
	}
	got, err := os.ReadFile(filepath.Join(targetDir, storage.SourceVideoFile))
	if err != nil {
		t.Fatalf("read copied video: %v", err)
	}
	if string(got) != "fake video bytes" {
		t.Errorf("copied video content = %q, want the fresh copy, not the stale leftover", got)
	}
}

func TestExportRunner_CancelCleansUpTempFile(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)

	seedExportFiles(t, st, p.ID)
	// Overwrite the video with something large enough that canceling
	// mid-copy (1 MiB chunks) is observable instead of finishing before
	// the cancel fires.
	videoPath, err := st.FilePath(p.ID, storage.SourceVideoFile)
	if err != nil {
		t.Fatalf("FilePath: %v", err)
	}
	if err := os.WriteFile(videoPath, make([]byte, 8<<20), 0o644); err != nil {
		t.Fatalf("seed big video: %v", err)
	}

	targetDir := filepath.Join(t.TempDir(), "MyProject")
	runner := NewExportRunner(repo, st)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	err = runner.Run(ctx, job.Job{ProjectID: p.ID, Type: job.TypeExport, Payload: targetDir}, func(float64, string) {})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run with canceled ctx: err = %v, want context.Canceled", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, storage.SourceVideoFile+exportTempSuffix)); !os.IsNotExist(err) {
		t.Error("canceled export left a temp file behind")
	}
	if _, err := os.Stat(filepath.Join(targetDir, storage.SourceVideoFile)); !os.IsNotExist(err) {
		t.Error("canceled export left a final (non-temp) file behind")
	}
}
