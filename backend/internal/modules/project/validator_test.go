package project

import (
	"bytes"
	"encoding/json"
	"testing"
)

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

func TestExportFileNamesFor(t *testing.T) {
	cases := []struct {
		name, project, video string
		want                 ExportFiles
	}{
		{"plain name", "MPL Game 3", "source.mp4", ExportFiles{"MPL Game 3.mp4", "MPL Game 3.highlight.json", "MPL Game 3.narasi.txt"}},
		{"extension follows original video", "MPL Game 3", "source.mkv", ExportFiles{"MPL Game 3.mkv", "MPL Game 3.highlight.json", "MPL Game 3.narasi.txt"}},
		{"no video file name yet", "MPL Game 3", "", ExportFiles{"MPL Game 3.mp4", "MPL Game 3.highlight.json", "MPL Game 3.narasi.txt"}},
		{"windows forbidden chars", `GEEK vs BTR: G2/3?`, "source.mp4", ExportFiles{"GEEK vs BTR_ G2_3_.mp4", "GEEK vs BTR_ G2_3_.highlight.json", "GEEK vs BTR_ G2_3_.narasi.txt"}},
		{"trailing dot trimmed", "Final. ", "source.mp4", ExportFiles{"Final.mp4", "Final.highlight.json", "Final.narasi.txt"}},
		{"empty name", "  ", "source.mp4", ExportFiles{"project.mp4", "project.highlight.json", "project.narasi.txt"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExportFileNamesFor(tc.project, tc.video); got != tc.want {
				t.Errorf("ExportFileNamesFor(%q, %q) = %+v, want %+v", tc.project, tc.video, got, tc.want)
			}
		})
	}
}

func TestRewriteHighlightVideo(t *testing.T) {
	t.Run("replaces video, keeps other fields and order", func(t *testing.T) {
		in := `{"game":"mlbb","draft":{"tim_a":{"nama":"A","pick":["Fanny"],"ban":[]}},"segmen":[{"mulai":"00:00:01"}],"video":"source.mp4","durasi":12.5}`
		got, err := RewriteHighlightVideo([]byte(in), `GEEK "A".mp4`)
		if err != nil {
			t.Fatalf("RewriteHighlightVideo: %v", err)
		}
		var compact bytes.Buffer
		if err := json.Compact(&compact, got); err != nil {
			t.Fatalf("output is not valid JSON: %v\n%s", err, got)
		}
		want := `{"game":"mlbb","draft":{"tim_a":{"nama":"A","pick":["Fanny"],"ban":[]}},"segmen":[{"mulai":"00:00:01"}],"video":"GEEK \"A\".mp4","durasi":12.5}`
		if compact.String() != want {
			t.Errorf("got  %s\nwant %s", compact.String(), want)
		}
	})
	t.Run("adds video when missing", func(t *testing.T) {
		got, err := RewriteHighlightVideo([]byte(`{"game":"mlbb"}`), "X.mp4")
		if err != nil {
			t.Fatalf("RewriteHighlightVideo: %v", err)
		}
		var v struct{ Video string }
		if err := json.Unmarshal(got, &v); err != nil || v.Video != "X.mp4" {
			t.Errorf("got %s (err %v), want video X.mp4", got, err)
		}
	})
	t.Run("rejects non-object", func(t *testing.T) {
		if _, err := RewriteHighlightVideo([]byte(`[1,2]`), "X.mp4"); err == nil {
			t.Error("want error for a JSON array")
		}
	})
}
