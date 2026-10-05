package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"

	"smeditor/internal/tools"
)

// Thumbnail grabs a single frame from videoPath at atSec seconds and
// writes it, scaled to 480px wide, to outPath, per docs/prd.md.
func (Ffmpeg) Thumbnail(ctx context.Context, ffmpegPath, videoPath, outPath string, atSec float64) error {
	resolved, found := tools.ResolveExecutable(ffmpegPath)
	if !found {
		return fmt.Errorf("ffmpeg tidak ditemukan di %s", ffmpegPath)
	}

	args := []string{
		"-hide_banner", "-nostdin", "-y",
		"-ss", strconv.FormatFloat(atSec, 'f', 2, 64),
		"-i", videoPath,
		"-vframes", "1",
		"-vf", "scale=480:-1",
		outPath,
	}
	out, err := exec.CommandContext(ctx, resolved, args...).CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg gagal membuat thumbnail: %s", string(out))
	}
	return nil
}
