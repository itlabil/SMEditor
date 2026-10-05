package ffmpeg

import "testing"

func TestParseVersion(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   string
	}{
		{
			"ffmpeg output",
			"ffmpeg version 6.1.1-3ubuntu5 Copyright (c) 2000-2023 the FFmpeg developers\nbuilt with gcc 13\n",
			"6.1.1-3ubuntu5",
		},
		{
			"ffprobe output",
			"ffprobe version 6.1.1-3ubuntu5 Copyright (c) 2007-2023 the FFmpeg developers\n",
			"6.1.1-3ubuntu5",
		},
		{"no version token", "command not found\n", ""},
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
