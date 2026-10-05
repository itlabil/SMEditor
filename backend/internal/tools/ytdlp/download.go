// Package ytdlp calls the yt-dlp binary.
package ytdlp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"smeditor/internal/tools"
)

// ProgressFunc reports download progress as it happens.
type ProgressFunc func(percent float64, message string)

type DownloadOptions struct {
	URL string
	// OutputPath is the final absolute destination, e.g. .../source.mp4.
	OutputPath string
	// FFmpegPath is the resolved ffmpeg binary, passed as
	// --ffmpeg-location so yt-dlp can mux without relying on PATH.
	FFmpegPath string
}

const tmpBasename = "download_tmp"

// Download runs yt-dlp with the format priority from docs/prd.md: H.264
// video + AAC audio in MP4, 1080p then 720p; falling back to the best
// format up to 1080p (which may be VP9 or AV1), then to the best format of
// any kind. The result is written to a temporary name in the destination
// folder and renamed to OutputPath only once the download succeeds, per
// .agents/skills/sm-external-tools.
func (Client) Download(ctx context.Context, ytdlpPath string, opts DownloadOptions, report ProgressFunc) error {
	resolved, found := tools.ResolveExecutable(ytdlpPath)
	if !found {
		return fmt.Errorf("yt-dlp tidak ditemukan di %s", ytdlpPath)
	}

	// --ffmpeg-location needs an actual path (file or containing folder);
	// a bare command name meant for PATH lookup (the default setting,
	// "ffmpeg") makes yt-dlp silently skip the merge step instead of
	// erroring, so it must be resolved the same way exec would resolve it.
	resolvedFFmpeg := opts.FFmpegPath
	if resolvedFFmpeg != "" {
		resolvedPath, found := tools.ResolveExecutable(opts.FFmpegPath)
		if !found {
			return fmt.Errorf("ffmpeg tidak ditemukan di %s", opts.FFmpegPath)
		}
		resolvedFFmpeg = resolvedPath
	}

	dir := filepath.Dir(opts.OutputPath)
	tmpl := filepath.Join(dir, tmpBasename+".%(ext)s")
	finalTmp := filepath.Join(dir, tmpBasename+".mp4")
	defer cleanupTemp(dir)

	args := []string{
		"--no-playlist",
		"--newline",
		"-f", "bv*[vcodec^=avc1][height<=1080]+ba[acodec^=mp4a]/bv*[vcodec^=avc1][height<=720]+ba[acodec^=mp4a]/b[height<=1080]/b",
		"--merge-output-format", "mp4",
		"--progress-template", progressTemplate,
	}
	if resolvedFFmpeg != "" {
		args = append(args, "--ffmpeg-location", resolvedFFmpeg)
	}
	args = append(args, "-o", tmpl, "--", opts.URL)

	cmd := exec.CommandContext(ctx, resolved, args...)
	tools.ConfigureProcessGroup(cmd)

	var tailLines []string
	err := tools.RunStreaming(cmd,
		func(line string) {
			if p, ok := parseProgressLine(line); ok {
				report(p.Percent, p.Message())
			}
		},
		func(line string) {
			tailLines = appendTail(tailLines, line)
		},
	)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("yt-dlp gagal mengunduh: %s", strings.Join(tailLines, "\n"))
	}

	if err := os.Rename(finalTmp, opts.OutputPath); err != nil {
		return fmt.Errorf("pindahkan hasil unduhan: %w", err)
	}
	return nil
}

// Title fetches a video's title with a separate, download-free call, per
// .agents/skills/sm-external-tools.
func (Client) Title(ctx context.Context, ytdlpPath, url string) (string, error) {
	resolved, found := tools.ResolveExecutable(ytdlpPath)
	if !found {
		return "", fmt.Errorf("yt-dlp tidak ditemukan di %s", ytdlpPath)
	}
	out, err := exec.CommandContext(ctx, resolved, "--no-playlist", "--skip-download", "--print", "title", "--", url).Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("ambil judul video: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func appendTail(tail []string, line string) []string {
	const maxLines = 20
	tail = append(tail, line)
	if len(tail) > maxLines {
		tail = tail[len(tail)-maxLines:]
	}
	return tail
}

// cleanupTemp removes any half-finished download_tmp.* file left behind
// by a canceled or failed download, per .agents/skills/sm-external-tools.
// It is harmless to call after a successful download: the temp file has
// already been renamed away by then.
func cleanupTemp(dir string) {
	matches, _ := filepath.Glob(filepath.Join(dir, tmpBasename+".*"))
	for _, m := range matches {
		os.Remove(m)
	}
}
