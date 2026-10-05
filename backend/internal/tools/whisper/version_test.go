package whisper

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   string
	}{
		{"simple version", "whisper.cpp 1.6.0\n", "whisper.cpp 1.6.0"},
		{"with extra lines", "whisper.cpp 1.6.0\nusage: whisper-cli [options]\n", "whisper.cpp 1.6.0"},
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
