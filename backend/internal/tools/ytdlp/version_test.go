package ytdlp

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   string
	}{
		{"simple version", "2024.08.06\n", "2024.08.06"},
		{"with debug lines", "2024.08.06\n[debug] Python 3.11.4\n[debug] exe versions: ffmpeg 6.0\n", "2024.08.06"},
		{"no trailing newline", "2024.12.23", "2024.12.23"},
		{"empty output", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseVersion(tc.output); got != tc.want {
				t.Errorf("parseVersion(%q) = %q, want %q", tc.output, got, tc.want)
			}
		})
	}
}
