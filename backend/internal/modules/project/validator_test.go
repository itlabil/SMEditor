package project

import "testing"

func TestIsYoutubeURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"watch url", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"bare domain", "https://youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"mobile domain", "https://m.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"short link", "https://youtu.be/dQw4w9WgXcQ", true},
		{"http scheme", "http://youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"music domain", "https://music.youtube.com/watch?v=dQw4w9WgXcQ", true},
		{"other domain", "https://vimeo.com/12345", false},
		{"lookalike domain", "https://youtube.com.evil.com/watch?v=1", false},
		{"lookalike subdomain", "https://notyoutube.com/watch?v=1", false},
		{"no scheme", "youtube.com/watch?v=1", false},
		{"javascript scheme", "javascript:alert(1)", false},
		{"empty", "", false},
		{"whitespace padded", "  https://youtu.be/dQw4w9WgXcQ  ", true},
		{"garbage", "not a url at all", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsYoutubeURL(tc.url); got != tc.want {
				t.Errorf("IsYoutubeURL(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}
