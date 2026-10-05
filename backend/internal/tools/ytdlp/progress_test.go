package ytdlp

import "testing"

func TestParseProgressLine(t *testing.T) {
	cases := []struct {
		name string
		line string
		want Progress
		ok   bool
	}{
		{
			name: "mid-download video stream",
			line: "SMPROGRESS  26.9%|   5.00MiB/s|00:05",
			want: Progress{Percent: 26.9, Speed: "5.00MiB/s", ETA: "00:05"},
			ok:   true,
		},
		{
			name: "complete with unknown eta",
			line: "SMPROGRESS 100.0%|4.00MiB/s|NA",
			want: Progress{Percent: 100.0, Speed: "4.00MiB/s", ETA: "NA"},
			ok:   true,
		},
		{
			name: "just started, unknown speed",
			line: "SMPROGRESS   0.0%| Unknown B/s|Unknown",
			want: Progress{Percent: 0.0, Speed: "Unknown B/s", ETA: "Unknown"},
			ok:   true,
		},
		{
			name: "merger message",
			line: `[Merger] Merging formats into "test.mp4"`,
			ok:   false,
		},
		{
			name: "destination message",
			line: "[download] Destination: test.f140.m4a",
			ok:   false,
		},
		{
			name: "delete original file message",
			line: "Deleting original file test.f137.mp4 (pass -k to keep)",
			ok:   false,
		},
		{
			name: "empty line",
			line: "",
			ok:   false,
		},
		{
			name: "prefix with no fields",
			line: "SMPROGRESS garbage",
			ok:   false,
		},
		{
			name: "prefix with unparsable percent",
			line: "SMPROGRESS abc%|1MiB/s|00:01",
			ok:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseProgressLine(tc.line)
			if ok != tc.ok {
				t.Fatalf("parseProgressLine(%q) ok = %v, want %v", tc.line, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("parseProgressLine(%q) = %+v, want %+v", tc.line, got, tc.want)
			}
		})
	}
}
