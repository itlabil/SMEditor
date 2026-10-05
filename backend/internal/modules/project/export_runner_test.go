package project

import (
	"context"
	"encoding/json"
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

	// Video and narasi are copied byte-for-byte under project-named file
	// names; highlight.json is checked separately (its "video" changes).
	copies := map[string]string{
		storage.SourceVideoFile: "MPL Game 3.mp4",
		storage.NarrationFile:   "MPL Game 3.narasi.txt",
	}
	for src, dst := range copies {
		srcPath, err := st.FilePath(p.ID, src)
		if err != nil {
			t.Fatalf("FilePath(%s): %v", src, err)
		}
		want, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("read source %s: %v", src, err)
		}
		got, err := os.ReadFile(filepath.Join(targetDir, dst))
		if err != nil {
			t.Fatalf("read copied %s: %v", dst, err)
		}
		if string(got) != string(want) {
			t.Errorf("copied %s content = %q, want %q", dst, got, want)
		}
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		t.Fatalf("read target dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := []string{"MPL Game 3.highlight.json", "MPL Game 3.mp4", "MPL Game 3.narasi.txt"}
	if strings.Join(names, "|") != strings.Join(want, "|") {
		t.Errorf("target dir files = %v, want exactly %v (no temp files, no fixed names)", names, want)
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

	highlightBytes, err := os.ReadFile(filepath.Join(targetDir, "MPL Game 3.highlight.json"))
	if err != nil {
		t.Fatalf("read copied highlight: %v", err)
	}
	var copied struct {
		Video  string  `json:"video"`
		Durasi float64 `json:"durasi"`
	}
	if err := json.Unmarshal(highlightBytes, &copied); err != nil {
		t.Fatalf("copied highlight is not valid JSON: %v", err)
	}
	if copied.Video != "MPL Game 3.mp4" || copied.Durasi != 12 {
		t.Errorf("copied highlight = %+v, want video %q and durasi kept", copied, "MPL Game 3.mp4")
	}
	if _, err := os.Stat(filepath.Join(targetDir, copied.Video)); err != nil {
		t.Errorf("copied video not found under the name the highlight references: %v", err)
	}

	// The project folder itself keeps its fixed names and content.
	srcPath, _ := st.FilePath(p.ID, storage.HighlightFile)
	if src, _ := os.ReadFile(srcPath); string(src) != `{"video":"source.mp4","durasi":12}` {
		t.Errorf("data/projects highlight.json changed: %s", src)
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
	stale := filepath.Join(targetDir, "MPL Game 3.mp4"+exportTempSuffix)
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
	got, err := os.ReadFile(filepath.Join(targetDir, "MPL Game 3.mp4"))
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

	if _, err := os.Stat(filepath.Join(targetDir, "MPL Game 3.mp4"+exportTempSuffix)); !os.IsNotExist(err) {
		t.Error("canceled export left a temp file behind")
	}
	if _, err := os.Stat(filepath.Join(targetDir, "MPL Game 3.mp4")); !os.IsNotExist(err) {
		t.Error("canceled export left a final (non-temp) file behind")
	}
}

func TestExportRunner_WindowsForbiddenCharsInProjectName(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	svc := NewService(repo, st, &fakeJobs{}, &fakePromptAssembler{text: "prompt palsu"}, &fakeFolderOpener{}, &fakeSettingsWriter{})
	req := validRequest()
	req.Name = `GEEK vs BTR: Game 2/3 "Final"?`
	p, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	seedExportFiles(t, st, p.ID)

	targetDir := filepath.Join(t.TempDir(), SanitizeFolderName(p.Name))
	if err := NewExportRunner(repo, st).Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeExport, Payload: targetDir}, func(float64, string) {}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	base := "GEEK vs BTR_ Game 2_3 _Final__"
	for _, name := range []string{base + ".mp4", base + ".highlight.json", base + ".narasi.txt"} {
		if _, err := os.Stat(filepath.Join(targetDir, name)); err != nil {
			t.Errorf("expected exported file %q: %v", name, err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(targetDir, base+".highlight.json"))
	if !strings.Contains(string(b), `"video": "`+base+`.mp4"`) {
		t.Errorf("highlight video field does not name %q: %s", base+".mp4", b)
	}
}
