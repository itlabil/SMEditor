package project

import (
	"net/url"
	"path/filepath"
	"strings"
)

var youtubeHosts = map[string]bool{
	"youtube.com":       true,
	"www.youtube.com":   true,
	"m.youtube.com":     true,
	"music.youtube.com": true,
	"youtu.be":          true,
	"www.youtu.be":      true,
}

// IsYoutubeURL reports whether raw is a well-formed http(s) URL on a
// YouTube domain, per .agents/rules/conventions.md ("Tolak URL selain
// domain YouTube sebelum memanggil yt-dlp").
func IsYoutubeURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Hostname() == "" {
		return false
	}
	return youtubeHosts[strings.ToLower(u.Hostname())]
}

// windowsForbiddenChars are the characters Windows forbids in a single
// path segment (NTFS/FAT). Linux only forbids "/" and the null byte,
// both already covered: "/" is in this set and a null byte is a control
// character, handled separately below.
const windowsForbiddenChars = `<>:"/\|?*`

// SanitizeFolderName turns name into a string safe to use as a single
// folder name on both Windows and Linux, per docs/prd.md ("Salin ke
// folder" creates a subfolder named after the project). Control
// characters and the characters Windows forbids become "_"; leading and
// trailing dots and spaces are trimmed, since Windows silently strips a
// trailing one and a leading one looks like a hidden file on Linux. A
// name that sanitizes to nothing falls back to "project".
func SanitizeFolderName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 0x20:
			b.WriteRune('_')
		case strings.ContainsRune(windowsForbiddenChars, r):
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	cleaned := strings.Trim(b.String(), " .")
	if cleaned == "" {
		return "project"
	}
	return cleaned
}

// ValidExportDest reports whether raw is well-formed enough to use as the
// destination for "Salin ke folder": non-empty and an absolute path. This
// is a pure, filesystem-free check; Service.ExportToFolder additionally
// verifies the path actually exists and is writable, which needs the
// filesystem.
func ValidExportDest(raw string) bool {
	return raw != "" && filepath.IsAbs(raw)
}
