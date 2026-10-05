package project

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"smeditor/internal/modules/job"
	"smeditor/internal/modules/settings"
	"smeditor/internal/storage"
	"smeditor/internal/tools/whisper"
)

type fakeAudioExtractor struct {
	called bool
	err    error
}

func (f *fakeAudioExtractor) ExtractAudio(ctx context.Context, ffmpegPath, videoPath, outPath string) error {
	f.called = true
	return f.err
}

type fakeWhisper struct {
	err       error
	rawOutput []byte
	lastOpts  whisper.TranscribeOptions
}

func (f *fakeWhisper) Transcribe(ctx context.Context, whisperPath string, opts whisper.TranscribeOptions, report whisper.ProgressFunc) error {
	f.lastOpts = opts
	report(50, "membuat transcript")
	if f.err != nil {
		return f.err
	}
	return os.WriteFile(opts.OutputBase+".json", f.rawOutput, 0o644)
}

const sampleWhisperOutput = `{"result":{"language":"id"},"transcription":[{"offsets":{"from":0,"to":2000},"text":" halo semua"}]}`

func TestTranscribeRunner_Success_WritesTranscriptFiles(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)
	if err := repo.SaveDownloadMetadata(context.Background(), p.ID, DownloadMetadata{DurationSec: 2.0, VideoCodec: "h264"}); err != nil {
		t.Fatalf("seed download metadata: %v", err)
	}

	audio := &fakeAudioExtractor{}
	wh := &fakeWhisper{rawOutput: []byte(sampleWhisperOutput)}
	runner := NewTranscribeRunner(repo, st, &fakeSettings{values: map[string]string{}}, audio, wh)

	var gotPercent float64
	err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeTranscribe}, func(percent float64, message string) {
		gotPercent = percent
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !audio.called {
		t.Error("ExtractAudio was not called")
	}
	if gotPercent != 50 {
		t.Errorf("reported percent = %v, want 50", gotPercent)
	}

	jsonPath, _ := st.FilePath(p.ID, storage.TranscriptJSONFile)
	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read transcript.json: %v", err)
	}
	if !strings.Contains(string(jsonContent), `"teks": "halo semua"`) {
		t.Errorf("transcript.json = %s, want it to contain the segment text", jsonContent)
	}

	txtPath, _ := st.FilePath(p.ID, storage.TranscriptTXTFile)
	txtContent, err := os.ReadFile(txtPath)
	if err != nil {
		t.Fatalf("read transcript.txt: %v", err)
	}
	if string(txtContent) != "[00:00:00] halo semua\n" {
		t.Errorf("transcript.txt = %q, want %q", txtContent, "[00:00:00] halo semua\n")
	}

	// The raw whisper working file must not be left behind.
	rawPath, _ := st.FilePath(p.ID, whisperRawOutputBase)
	if _, err := os.Stat(rawPath + ".json"); !os.IsNotExist(err) {
		t.Error("raw whisper output file was not cleaned up")
	}
}

func TestTranscribeRunner_DefaultsLanguageToAuto(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st) // TranscriptLang is "" by default

	wh := &fakeWhisper{rawOutput: []byte(sampleWhisperOutput)}
	runner := NewTranscribeRunner(repo, st, &fakeSettings{values: map[string]string{}}, &fakeAudioExtractor{}, wh)

	if err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeTranscribe}, func(float64, string) {}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if wh.lastOpts.Language != "auto" {
		t.Errorf("Language passed to whisper = %q, want auto", wh.lastOpts.Language)
	}
}

func TestTranscribeRunner_UsesRequestedLanguage(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)
	if err := repo.UpdateTranscriptLang(context.Background(), p.ID, "id"); err != nil {
		t.Fatalf("UpdateTranscriptLang: %v", err)
	}

	wh := &fakeWhisper{rawOutput: []byte(sampleWhisperOutput)}
	runner := NewTranscribeRunner(repo, st, &fakeSettings{values: map[string]string{}}, &fakeAudioExtractor{}, wh)

	if err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeTranscribe}, func(float64, string) {}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if wh.lastOpts.Language != "id" {
		t.Errorf("Language passed to whisper = %q, want id", wh.lastOpts.Language)
	}
}

func TestTranscribeRunner_WhisperDeviceSettingControlsNoGPU(t *testing.T) {
	cases := []struct {
		device string
		noGPU  bool
	}{
		{"cpu", true},
		{"gpu", false},
		{"auto", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.device, func(t *testing.T) {
			repo, st := newTestRepoAndStorage(t)
			p := seedTestProject(t, repo, st)

			wh := &fakeWhisper{rawOutput: []byte(sampleWhisperOutput)}
			runner := NewTranscribeRunner(repo, st, &fakeSettings{values: map[string]string{settings.KeyWhisperDevice: tc.device}}, &fakeAudioExtractor{}, wh)

			if err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeTranscribe}, func(float64, string) {}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			if wh.lastOpts.NoGPU != tc.noGPU {
				t.Errorf("NoGPU = %v, want %v for whisper_device=%q", wh.lastOpts.NoGPU, tc.noGPU, tc.device)
			}
		})
	}
}

func TestTranscribeRunner_ExtractAudioError_NeverCallsWhisper(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)

	audio := &fakeAudioExtractor{err: errors.New("ffmpeg gagal mengekstrak audio")}
	wh := &fakeWhisper{rawOutput: []byte(sampleWhisperOutput)}
	runner := NewTranscribeRunner(repo, st, &fakeSettings{values: map[string]string{}}, audio, wh)

	err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeTranscribe}, func(float64, string) {})
	if err == nil {
		t.Fatal("Run with failing audio extraction: want error, got nil")
	}
	if wh.lastOpts.AudioPath != "" {
		t.Error("Transcribe was called even though ExtractAudio failed")
	}
}

func TestTranscribeRunner_WhisperError_NoFilesWritten(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)

	wh := &fakeWhisper{err: errors.New("whisper gagal: model tidak ditemukan")}
	runner := NewTranscribeRunner(repo, st, &fakeSettings{values: map[string]string{}}, &fakeAudioExtractor{}, wh)

	err := runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeTranscribe}, func(float64, string) {})
	if err == nil {
		t.Fatal("Run with failing whisper: want error, got nil")
	}

	jsonPath, _ := st.FilePath(p.ID, storage.TranscriptJSONFile)
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Error("transcript.json should not exist after a failed whisper run")
	}
}
