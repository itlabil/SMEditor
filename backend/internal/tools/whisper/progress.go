package whisper

import (
	"strconv"
	"strings"
)

const progressMarker = "progress ="

// parseProgressLine parses whisper-cli's --print-progress stderr output,
// e.g. "whisper_print_progress_callback: progress =  39%". Lines that
// don't match (init logs, segment text, ...) return ok=false and must be
// ignored, not treated as an error, per .agents/skills/sm-external-tools.
func parseProgressLine(line string) (percent float64, ok bool) {
	idx := strings.Index(line, progressMarker)
	if idx < 0 {
		return 0, false
	}
	rest := strings.TrimSpace(line[idx+len(progressMarker):])
	rest = strings.TrimSuffix(rest, "%")
	p, err := strconv.ParseFloat(strings.TrimSpace(rest), 64)
	if err != nil {
		return 0, false
	}
	return p, true
}
