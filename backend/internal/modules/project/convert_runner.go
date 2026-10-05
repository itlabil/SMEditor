package project

import (
	"context"
	"fmt"
	"os"

	"smeditor/internal/modules/job"
	"smeditor/internal/modules/settings"
	"smeditor/internal/storage"
	"smeditor/internal/tools/ffmpeg"
)

// ConverterClient re-encodes a video to H.264. Implemented by
// tools/ffmpeg.Ffmpeg.
type ConverterClient interface {
	ConvertToH264(ctx context.Context, ffmpegPath, inPath, outPath string, durationSec float64, report ffmpeg.ProgressFunc) error
}

// ConvertRunner implements job.Runner for job.TypeConvert: it re-encodes
// a project's source video to H.264 in place, for sources yt-dlp could
// only provide as VP9/AV1, per docs/prd.md.
type ConvertRunner struct {
	repo     *Repository
	storage  *storage.Storage
	settings SettingsReader
	ffmpeg   ConverterClient
	ffprobe  FfprobeClient
}

func NewConvertRunner(
	repo *Repository,
	st *storage.Storage,
	settingsReader SettingsReader,
	ffmpegClient ConverterClient,
	ffprobeClient FfprobeClient,
) *ConvertRunner {
	return &ConvertRunner{repo: repo, storage: st, settings: settingsReader, ffmpeg: ffmpegClient, ffprobe: ffprobeClient}
}

func (r *ConvertRunner) Type() string { return job.TypeConvert }

func (r *ConvertRunner) Run(ctx context.Context, j job.Job, report job.ProgressFunc) error {
	p, err := r.repo.FindByID(ctx, j.ProjectID)
	if err != nil {
		return fmt.Errorf("baca project: %w", err)
	}

	cfg, err := r.settings.Get(ctx)
	if err != nil {
		return fmt.Errorf("baca pengaturan: %w", err)
	}

	srcPath, err := r.storage.FilePath(j.ProjectID, storage.SourceVideoFile)
	if err != nil {
		return err
	}
	tmpPath, err := r.storage.FilePath(j.ProjectID, storage.ConvertTempFile)
	if err != nil {
		return err
	}

	err = r.ffmpeg.ConvertToH264(ctx, cfg[settings.KeyFfmpegPath], srcPath, tmpPath, p.DurationSec, func(percent float64, message string) {
		report(percent, message)
	})
	if err != nil {
		return err
	}

	if err := os.Rename(tmpPath, srcPath); err != nil {
		return fmt.Errorf("timpa video dengan hasil konversi: %w", err)
	}

	meta, err := r.ffprobe.Probe(ctx, cfg[settings.KeyFfprobePath], srcPath)
	if err != nil {
		return fmt.Errorf("baca metadata setelah konversi: %w", err)
	}
	if err := r.repo.UpdateVideoCodecAndSize(ctx, j.ProjectID, meta.VideoCodec, meta.SizeBytes); err != nil {
		return fmt.Errorf("simpan metadata setelah konversi: %w", err)
	}
	return nil
}
