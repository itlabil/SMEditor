package project

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// ExportFileNamesFor names the exported files after the project, using
// the same SanitizeFolderName as the subfolder (SM-17): "<nama><ext>"
// with the original video's extension, "<nama>.highlight.json", and
// "<nama>.narasi.txt".
func ExportFileNamesFor(projectName, videoFile string) ExportFiles {
	base := SanitizeFolderName(projectName)
	ext := filepath.Ext(videoFile)
	if ext == "" {
		ext = ".mp4"
	}
	return ExportFiles{
		Video:     base + ext,
		Highlight: base + ".highlight.json",
		Narasi:    base + ".narasi.txt",
	}
}

// RewriteHighlightVideo returns highlight.json content with its top-level
// "video" field set to video (appended if missing), keeping every other
// field and their order intact, so the exported copy points at the
// renamed video file.
func RewriteHighlightVideo(data []byte, video string) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, fmt.Errorf("highlight.json bukan objek JSON")
	}

	videoJSON, err := json.Marshal(video)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	out.WriteByte('{')
	replaced := false
	for i := 0; dec.More(); i++ {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("baca highlight.json: %w", err)
		}
		key, _ := tok.(string)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, fmt.Errorf("baca highlight.json: %w", err)
		}
		if key == "video" {
			value, replaced = videoJSON, true
		}
		if i > 0 {
			out.WriteByte(',')
		}
		keyJSON, _ := json.Marshal(key)
		out.Write(keyJSON)
		out.WriteByte(':')
		out.Write(value)
	}
	if !replaced {
		if out.Len() > 1 {
			out.WriteByte(',')
		}
		out.WriteString(`"video":`)
		out.Write(videoJSON)
	}
	out.WriteByte('}')

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, out.Bytes(), "", "  "); err != nil {
		return nil, fmt.Errorf("susun highlight.json: %w", err)
	}
	return pretty.Bytes(), nil
}

// ValidExportDest reports whether raw is well-formed enough to use as the
// destination for "Salin ke folder": non-empty and an absolute path. This
// is a pure, filesystem-free check; Service.ExportToFolder additionally
// verifies the path actually exists and is writable, which needs the
// filesystem.
func ValidExportDest(raw string) bool {
	return raw != "" && filepath.IsAbs(raw)
}
