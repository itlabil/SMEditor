package project

import (
	"context"
	"errors"
	"os"
	"testing"

	"smeditor/internal/modules/job"
	"smeditor/internal/storage"
	"smeditor/internal/tools/ffmpeg"
)

type fakeConverter struct {
	err        error
	writeBytes []byte
}

func (f *fakeConverter) ConvertToH264(ctx context.Context, ffmpegPath, inPath, outPath string, durationSec float64, report ffmpeg.ProgressFunc) error {
	report(50, "mengonversi")
	if f.err != nil {
		return f.err
	}
	return os.WriteFile(outPath, f.writeBytes, 0o644)
}

func TestConvertRunner_Success_RenamesAndUpdatesMetadata(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)

	srcPath, err := st.FilePath(p.ID, storage.SourceVideoFile)
	if err != nil {
		t.Fatalf("FilePath: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte("original vp9 bytes"), 0o644); err != nil {
		t.Fatalf("seed source file: %v", err)
	}

	conv := &fakeConverter{writeBytes: []byte("converted h264 bytes")}
	probe := &fakeFfprobe{meta: ffmpeg.Metadata{VideoCodec: "h264", SizeBytes: int64(len(conv.writeBytes))}}
	runner := NewConvertRunner(repo, st, &fakeSettings{values: map[string]string{}}, conv, probe)

	var gotPercent float64
	err = runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeConvert}, func(percent float64, message string) {
		gotPercent = percent
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if gotPercent != 50 {
		t.Errorf("reported percent = %v, want 50", gotPercent)
	}

	content, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read source after convert: %v", err)
	}
	if string(content) != "converted h264 bytes" {
		t.Errorf("source.mp4 content = %q, want the converted file's content", content)
	}

	got, err := repo.FindByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.VideoCodec != "h264" || got.SizeBytes != int64(len(conv.writeBytes)) {
		t.Errorf("project not updated after convert: codec=%q size=%d", got.VideoCodec, got.SizeBytes)
	}
}

func TestConvertRunner_ConvertError_LeavesSourceUntouched(t *testing.T) {
	repo, st := newTestRepoAndStorage(t)
	p := seedTestProject(t, repo, st)

	srcPath, err := st.FilePath(p.ID, storage.SourceVideoFile)
	if err != nil {
		t.Fatalf("FilePath: %v", err)
	}
	if err := os.WriteFile(srcPath, []byte("original vp9 bytes"), 0o644); err != nil {
		t.Fatalf("seed source file: %v", err)
	}

	conv := &fakeConverter{err: errors.New("ffmpeg gagal konversi: codec not supported")}
	runner := NewConvertRunner(repo, st, &fakeSettings{values: map[string]string{}}, conv, &fakeFfprobe{})

	err = runner.Run(context.Background(), job.Job{ProjectID: p.ID, Type: job.TypeConvert}, func(float64, string) {})
	if err == nil {
		t.Fatal("Run with failing convert: want error, got nil")
	}

	content, readErr := os.ReadFile(srcPath)
	if readErr != nil {
		t.Fatalf("read source after failed convert: %v", readErr)
	}
	if string(content) != "original vp9 bytes" {
		t.Errorf("source.mp4 content = %q, want it untouched after a failed convert", content)
	}
}
