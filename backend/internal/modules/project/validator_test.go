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

func TestSanitizeFolderName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain name", "MPL Game 3", "MPL Game 3"},
		{"windows forbidden chars", `A<B>C:D"E/F\G|H?I*J`, "A_B_C_D_E_F_G_H_I_J"},
		{"control characters", "Line1\nLine2\tTabbed", "Line1_Line2_Tabbed"},
		{"trailing dot and space", "Final Boss. ", "Final Boss"},
		{"leading dot and space", " .Hidden", "Hidden"},
		{"only forbidden characters", `///***`, "______"},
		{"only whitespace", "   ", "project"},
		{"empty", "", "project"},
		{"unicode preserved", "Mabar Mobile Legends 五", "Mabar Mobile Legends 五"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeFolderName(tc.in); got != tc.want {
				t.Errorf("SanitizeFolderName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestValidExportDest(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"empty", "", false},
		{"relative", "data/projects", false},
		{"relative with dots", "../outside", false},
		{"unix absolute", "/mnt/d/Premiere/MyProject", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidExportDest(tc.path); got != tc.want {
				t.Errorf("ValidExportDest(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
