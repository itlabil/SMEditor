package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"

	"smeditor/internal/tools"
)

// ExtractAudio pulls the audio track out of videoPath into a 16kHz mono
// PCM WAV file at outPath, the format Whisper expects, per docs/prd.md.
func (Ffmpeg) ExtractAudio(ctx context.Context, ffmpegPath, videoPath, outPath string) error {
	resolved, found := tools.ResolveExecutable(ffmpegPath)
	if !found {
		return fmt.Errorf("ffmpeg tidak ditemukan di %s", ffmpegPath)
	}

	args := []string{
		"-hide_banner", "-nostdin", "-y",
		"-i", videoPath,
		"-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le",
		outPath,
	}
	out, err := exec.CommandContext(ctx, resolved, args...).CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg gagal mengekstrak audio: %s", string(out))
	}
	return nil
}
