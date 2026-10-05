package ytdlp

import "strings"

// parseVersion keeps only the first line of `yt-dlp --version`, e.g.
// "2024.08.06\n[debug] Python 3.11 ...".
func parseVersion(output string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(output), "\n")
	return strings.TrimSpace(line)
}
