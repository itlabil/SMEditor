package project

import (
	"net/url"
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
