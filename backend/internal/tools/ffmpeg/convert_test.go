package ffmpeg

import "testing"

func TestPercentFromOutTimeUs(t *testing.T) {
	cases := []struct {
		name        string
		outTimeUs   int64
		durationSec float64
		want        float64
	}{
		{"halfway", 75_000_000, 150, 50},
		{"start", 0, 150, 0},
		{"complete", 150_000_000, 150, 100},
		{"slightly over due to rounding", 150_500_000, 150, 100},
		{"unknown duration", 75_000_000, 0, 0},
		{"negative duration", 75_000_000, -1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := percentFromOutTimeUs(tc.outTimeUs, tc.durationSec); got != tc.want {
				t.Errorf("percentFromOutTimeUs(%d, %v) = %v, want %v", tc.outTimeUs, tc.durationSec, got, tc.want)
			}
		})
	}
}
