package project

import (
	"context"
	"fmt"
	"strings"

	"smeditor/internal/modules/job"
	"smeditor/internal/modules/settings"
	"smeditor/internal/storage"
	"smeditor/internal/tools/ffmpeg"
	"smeditor/internal/tools/ytdlp"
)

// YtdlpClient downloads a video and reads its title. Implemented by
// tools/ytdlp.Client.
type YtdlpClient interface {
	Download(ctx context.Context, ytdlpPath string, opts ytdlp.DownloadOptions, report ytdlp.ProgressFunc) error
	Title(ctx context.Context, ytdlpPath, url string) (string, error)
}

// FfprobeClient reads a video file's metadata. Implemented by
// tools/ffmpeg.Ffprobe.
type FfprobeClient interface {
	Probe(ctx context.Context, ffprobePath, videoPath string) (ffmpeg.Metadata, error)
}

// ThumbnailClient grabs a thumbnail frame. Implemented by
// tools/ffmpeg.Ffmpeg.
type ThumbnailClient interface {
	Thumbnail(ctx context.Context, ffmpegPath, videoPath, outPath string, atSec float64) error
}

// SettingsReader reads the currently configured tool paths. Implemented
// by settings.Service.
type SettingsReader interface {
	Get(ctx context.Context) (map[string]string, error)
}

// DownloadRunner implements job.Runner for job.TypeDownload: it downloads
// a project's source video, probes its metadata, grabs a thumbnail, and
// queues whatever comes next (convert, if the codec isn't H.264, or
// straight to transcribe), per docs/flow.md section 4.2.
type DownloadRunner struct {
	repo     *Repository
	storage  *storage.Storage
	settings SettingsReader
	ytdlp    YtdlpClient
	ffprobe  FfprobeClient
	thumb    ThumbnailClient
	jobs     JobEnqueuer
}

func NewDownloadRunner(
	repo *Repository,
	st *storage.Storage,
	settingsReader SettingsReader,
	ytdlpClient YtdlpClient,
	ffprobeClient FfprobeClient,
	thumbClient ThumbnailClient,
	jobs JobEnqueuer,
) *DownloadRunner {
	return &DownloadRunner{
		repo:     repo,
		storage:  st,
		settings: settingsReader,
		ytdlp:    ytdlpClient,
		ffprobe:  ffprobeClient,
		thumb:    thumbClient,
		jobs:     jobs,
	}
}

func (r *DownloadRunner) Type() string { return job.TypeDownload }

func (r *DownloadRunner) Run(ctx context.Context, j job.Job, report job.ProgressFunc) error {
	p, err := r.repo.FindByID(ctx, j.ProjectID)
	if err != nil {
		return fmt.Errorf("baca project: %w", err)
	}

	cfg, err := r.settings.Get(ctx)
	if err != nil {
		return fmt.Errorf("baca pengaturan: %w", err)
	}

	title, err := r.ytdlp.Title(ctx, cfg[settings.KeyYtdlpPath], p.YoutubeURL)
	if err != nil {
		return err
	}

	videoPath, err := r.storage.FilePath(j.ProjectID, storage.SourceVideoFile)
	if err != nil {
		return err
	}

	err = r.ytdlp.Download(ctx, cfg[settings.KeyYtdlpPath], ytdlp.DownloadOptions{
		URL:        p.YoutubeURL,
		OutputPath: videoPath,
		FFmpegPath: cfg[settings.KeyFfmpegPath],
	}, func(percent float64, message string) {
		report(percent, message)
	})
	if err != nil {
		return err
	}

	meta, err := r.ffprobe.Probe(ctx, cfg[settings.KeyFfprobePath], videoPath)
	if err != nil {
		return fmt.Errorf("baca metadata video: %w", err)
	}

	thumbPath, err := r.storage.FilePath(j.ProjectID, storage.ThumbnailFile)
	if err != nil {
		return err
	}
	if err := r.thumb.Thumbnail(ctx, cfg[settings.KeyFfmpegPath], videoPath, thumbPath, meta.DurationSec*0.1); err != nil {
		return fmt.Errorf("buat thumbnail: %w", err)
	}

	if err := r.repo.SaveDownloadMetadata(ctx, j.ProjectID, DownloadMetadata{
		VideoTitle:  title,
		DurationSec: meta.DurationSec,
		Width:       meta.Width,
		Height:      meta.Height,
		FPS:         meta.FPS,
		VideoCodec:  meta.VideoCodec,
		VideoFile:   storage.SourceVideoFile,
		SizeBytes:   meta.SizeBytes,
	}); err != nil {
		return fmt.Errorf("simpan metadata video: %w", err)
	}

	next := job.TypeTranscribe
	if !strings.EqualFold(meta.VideoCodec, "h264") {
		next = job.TypeConvert
	}
	if _, err := r.jobs.Enqueue(ctx, j.ProjectID, next); err != nil {
		return fmt.Errorf("antrekan job %s: %w", next, err)
	}
	return nil
}
