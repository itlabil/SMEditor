package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"smeditor/internal/tools"
)

// ProgressFunc reports conversion progress as it happens.
type ProgressFunc func(percent float64, message string)

// ConvertToH264 re-encodes inPath to H.264 video / AAC audio at outPath,
// keeping the source resolution and fps untouched, per docs/prd.md
// ("Konversi ke H.264 ... resolusi dan fps tetap"). durationSec (from a
// prior Probe call) is used to turn ffmpeg's out_time_us into a
// percentage.
func (Ffmpeg) ConvertToH264(ctx context.Context, ffmpegPath, inPath, outPath string, durationSec float64, report ProgressFunc) error {
	resolved, found := tools.ResolveExecutable(ffmpegPath)
	if !found {
		return fmt.Errorf("ffmpeg tidak ditemukan di %s", ffmpegPath)
	}

	args := []string{
		"-hide_banner", "-nostdin", "-y",
		"-i", inPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-progress", "pipe:1",
		outPath,
	}
	cmd := exec.CommandContext(ctx, resolved, args...)
	tools.ConfigureProcessGroup(cmd)

	var tailLines []string
	var outTimeUs int64
	err := tools.RunStreaming(cmd,
		func(line string) {
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				return
			}
			switch key {
			case "out_time_us":
				if v, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
					outTimeUs = v
				}
			case "progress":
				report(percentFromOutTimeUs(outTimeUs, durationSec), "Mengonversi ke H.264")
			}
		},
		func(line string) {
			tailLines = appendTail(tailLines, line)
		},
	)
	if err != nil {
		os.Remove(outPath)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg gagal konversi: %s", strings.Join(tailLines, "\n"))
	}
	return nil
}

// percentFromOutTimeUs turns ffmpeg's -progress out_time_us (microseconds
// encoded so far) into a 0-100 percentage of durationSec. It returns 0 if
// durationSec is unknown (<= 0), since percent-of-unknown is meaningless.
func percentFromOutTimeUs(outTimeUs int64, durationSec float64) float64 {
	if durationSec <= 0 {
		return 0
	}
	percent := float64(outTimeUs) / 1e6 / durationSec * 100
	if percent > 100 {
		return 100
	}
	if percent < 0 {
		return 0
	}
	return percent
}

func appendTail(tail []string, line string) []string {
	const maxLines = 20
	tail = append(tail, line)
	if len(tail) > maxLines {
		tail = tail[len(tail)-maxLines:]
	}
	return tail
}
