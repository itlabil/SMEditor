package settings

import (
	"context"
	"path/filepath"
	"testing"

	"smeditor/internal/db"
)

type fakeToolClient struct {
	path    string
	found   bool
	version string
}

func (f *fakeToolClient) Check(ctx context.Context, path string) (string, bool, string) {
	f.path = path
	return path, f.found, f.version
}

type fakeWhisperClient struct {
	path    string
	found   bool
	backend string
	gpu     string
}

func (f *fakeWhisperClient) CheckWithBackend(ctx context.Context, path string) (string, bool, string, string, string) {
	f.path = path
	return path, f.found, "", f.backend, f.gpu
}

type fakeModelClient struct {
	path  string
	found bool
}

func (f *fakeModelClient) CheckModel(ctx context.Context, path string) (string, bool, string) {
	f.path = path
	return path, f.found, ""
}

func newTestService(t *testing.T) (*Service, *fakeToolClient, *fakeToolClient, *fakeToolClient, *fakeWhisperClient, *fakeModelClient) {
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
	ytdlp := &fakeToolClient{found: true, version: "2024.08.06"}
	ffmpeg := &fakeToolClient{found: true, version: "6.1.1"}
	ffprobe := &fakeToolClient{found: true, version: "6.1.1"}
	whisper := &fakeWhisperClient{found: false}
	whisperModel := &fakeModelClient{found: true}

	return NewService(repo, ytdlp, ffmpeg, ffprobe, whisper, whisperModel), ytdlp, ffmpeg, ffprobe, whisper, whisperModel
}

func TestServiceGetReturnsDefaults(t *testing.T) {
	svc, _, _, _, _, _ := newTestService(t)

	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	for key, want := range defaults {
		if got[key] != want {
			t.Errorf("Get()[%s] = %q, want default %q", key, got[key], want)
		}
	}
}

func TestServiceUpdateOverridesDefaultAndPersists(t *testing.T) {
	svc, _, _, _, _, _ := newTestService(t)
	ctx := context.Background()

	got, err := svc.Update(ctx, map[string]string{KeyYtdlpPath: "/opt/yt-dlp"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got[KeyYtdlpPath] != "/opt/yt-dlp" {
		t.Errorf("Update()[%s] = %q, want %q", KeyYtdlpPath, got[KeyYtdlpPath], "/opt/yt-dlp")
	}
	// other keys stay at their default
	if got[KeyFfmpegPath] != defaults[KeyFfmpegPath] {
		t.Errorf("Update() changed unrelated key %s to %q", KeyFfmpegPath, got[KeyFfmpegPath])
	}

	again, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if again[KeyYtdlpPath] != "/opt/yt-dlp" {
		t.Errorf("Get() after Update = %q, want persisted value", again[KeyYtdlpPath])
	}
}

func TestServiceUpdateRejectsUnknownKey(t *testing.T) {
	svc, _, _, _, _, _ := newTestService(t)

	_, err := svc.Update(context.Background(), map[string]string{"not_a_real_key": "x"})
	if err == nil {
		t.Fatal("Update with unknown key: want error, got nil")
	}
}

func TestServiceUpdateRejectsInvalidWhisperDevice(t *testing.T) {
	svc, _, _, _, _, _ := newTestService(t)

	_, err := svc.Update(context.Background(), map[string]string{KeyWhisperDevice: "tpu"})
	if err == nil {
		t.Fatal("Update with invalid whisper_device: want error, got nil")
	}
}

func TestServiceCheckReportsEachTool(t *testing.T) {
	svc, ytdlp, ffmpeg, ffprobe, whisper, whisperModel := newTestService(t)

	results, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("Check() returned %d results, want 5", len(results))
	}

	byTool := map[string]CheckResult{}
	for _, r := range results {
		byTool[r.Tool] = r
	}

	if !byTool["yt-dlp"].Found || byTool["yt-dlp"].Version != ytdlp.version {
		t.Errorf("yt-dlp result = %+v", byTool["yt-dlp"])
	}
	if !byTool["ffmpeg"].Found || byTool["ffmpeg"].Version != ffmpeg.version {
		t.Errorf("ffmpeg result = %+v", byTool["ffmpeg"])
	}
	if !byTool["ffprobe"].Found || byTool["ffprobe"].Version != ffprobe.version {
		t.Errorf("ffprobe result = %+v", byTool["ffprobe"])
	}
	if byTool["whisper"].Found {
		t.Errorf("whisper result = %+v, want found=false", byTool["whisper"])
	}
	if whisper.path != defaults[KeyWhisperPath] {
		t.Errorf("whisper client received path %q, want default %q", whisper.path, defaults[KeyWhisperPath])
	}

	modelResult, ok := byTool["model whisper"]
	if !ok {
		t.Fatal(`Check() has no "model whisper" result`)
	}
	if !modelResult.Found {
		t.Errorf("model whisper result = %+v, want found=true", modelResult)
	}
	if whisperModel.path != defaults[KeyWhisperModel] {
		t.Errorf("whisper model client received path %q, want default %q", whisperModel.path, defaults[KeyWhisperModel])
	}
}

func TestServiceCheckReportsModelNotFound(t *testing.T) {
	svc, _, _, _, _, whisperModel := newTestService(t)
	whisperModel.found = false

	results, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	byTool := map[string]CheckResult{}
	for _, r := range results {
		byTool[r.Tool] = r
	}
	if byTool["model whisper"].Found {
		t.Errorf(`model whisper result = %+v, want found=false`, byTool["model whisper"])
	}
}

func TestServiceCheckReportsWhisperBackend(t *testing.T) {
	cases := []struct {
		name, backend, gpu string
	}{
		{"gpu", "gpu", "NVIDIA GeForce GTX 1650"},
		{"cpu", "cpu", ""},
		{"unknown", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _, _, whisper, _ := newTestService(t)
			whisper.found, whisper.backend, whisper.gpu = true, tc.backend, tc.gpu

			results, err := svc.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			for _, r := range results {
				if r.Tool == "whisper" {
					if r.Backend != tc.backend || r.GPU != tc.gpu {
						t.Errorf("whisper result = %+v, want backend %q gpu %q", r, tc.backend, tc.gpu)
					}
				} else if r.Backend != "" {
					t.Errorf("%s result has backend %q, want none", r.Tool, r.Backend)
				}
			}
		})
	}
}
