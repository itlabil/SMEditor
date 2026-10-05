package ytdlp

import (
	"strconv"
	"strings"
)

const progressPrefix = "SMPROGRESS "

// progressTemplate is passed to yt-dlp's --progress-template so every
// progress line has a fixed, easy-to-parse shape, per
// .agents/skills/sm-external-tools.
const progressTemplate = "download:" + progressPrefix + "%(progress._percent_str)s|%(progress._speed_str)s|%(progress._eta_str)s"

// Progress is one parsed line of yt-dlp download progress.
type Progress struct {
	Percent float64
	Speed   string
	ETA     string
}

// Message renders a human-readable Indonesian progress line.
func (p Progress) Message() string {
	return "Mengunduh " + strconv.FormatFloat(p.Percent, 'f', 1, 64) + "% - " + p.Speed + " - sisa " + p.ETA
}

// parseProgressLine parses one line of yt-dlp output using progressTemplate.
// Lines that don't match (merger/delete messages, warnings, blank lines,
// ...) return ok=false and must be ignored, not treated as an error, per
// .agents/skills/sm-external-tools.
func parseProgressLine(line string) (Progress, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, progressPrefix) {
		return Progress{}, false
	}
	fields := strings.Split(strings.TrimPrefix(line, progressPrefix), "|")
	if len(fields) < 3 {
		return Progress{}, false
	}
	percentStr := strings.TrimSuffix(strings.TrimSpace(fields[0]), "%")
	percent, err := strconv.ParseFloat(percentStr, 64)
	if err != nil {
		return Progress{}, false
	}
	return Progress{
		Percent: percent,
		Speed:   strings.TrimSpace(fields[1]),
		ETA:     strings.TrimSpace(fields[2]),
	}, true
}
