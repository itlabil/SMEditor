package whisper

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"smeditor/internal/tools"
)

// ProgressFunc reports transcription progress as it happens.
type ProgressFunc func(percent float64, message string)

type TranscribeOptions struct {
	AudioPath string
	// OutputBase is the output path without extension; whisper-cli writes
	// OutputBase+".json".
	OutputBase string
	ModelPath  string
	// Language is a whisper language code ("auto", "id", "en", "tl", ...),
	// per docs/prd.md.
	Language string
	// NoGPU forces CPU-only decoding, set when settings.whisper_device is
	// "cpu".
	NoGPU bool
}

// Transcribe runs whisper-cli against a 16kHz mono WAV file and writes
// its own JSON output to OutputBase+".json"; the caller (project module)
// converts that into transcript.json/transcript.txt.
func (Client) Transcribe(ctx context.Context, whisperPath string, opts TranscribeOptions, report ProgressFunc) error {
	resolvedBin, found := tools.ResolveExecutable(whisperPath)
	if !found {
		return fmt.Errorf("whisper tidak ditemukan di %s", whisperPath)
	}
	resolvedModel, found := tools.ResolveExecutable(opts.ModelPath)
	if !found {
		return fmt.Errorf("model whisper tidak ditemukan di %s", opts.ModelPath)
	}

	lang := opts.Language
	if lang == "" {
		lang = "auto"
	}

	args := []string{
		"-m", resolvedModel,
		"-f", opts.AudioPath,
		"-l", lang,
		"-oj",
		"-of", opts.OutputBase,
		"-pp",
		"-np",
		"-t", strconv.Itoa(defaultThreads()),
	}
	if opts.NoGPU {
		args = append(args, "-ng")
	}

	cmd := exec.CommandContext(ctx, resolvedBin, args...)
	tools.ConfigureProcessGroup(cmd)

	var tailLines []string
	err := tools.RunStreaming(cmd,
		func(line string) {}, // segment text on stdout; the JSON file is read afterward instead
		func(line string) {
			if percent, ok := parseProgressLine(line); ok {
				report(percent, "Membuat transcript")
				return
			}
			tailLines = appendTail(tailLines, line)
		},
	)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("whisper gagal: %s", strings.Join(tailLines, "\n"))
	}
	return nil
}

// defaultThreads is the physical core count, capped at 8, per
// .agents/skills/sm-external-tools.
func defaultThreads() int {
	n := runtime.NumCPU()
	if n > 8 {
		return 8
	}
	if n < 1 {
		return 1
	}
	return n
}

func appendTail(tail []string, line string) []string {
	const maxLines = 20
	tail = append(tail, line)
	if len(tail) > maxLines {
		tail = tail[len(tail)-maxLines:]
	}
	return tail
}
