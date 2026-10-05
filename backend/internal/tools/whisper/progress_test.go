package whisper

import "testing"

func TestParseProgressLine(t *testing.T) {
	cases := []struct {
		name string
		line string
		want float64
		ok   bool
	}{
		{"typical", "whisper_print_progress_callback: progress =  39%", 39, true},
		{"complete", "whisper_print_progress_callback: progress = 100%", 100, true},
		{"zero padded differently", "whisper_print_progress_callback: progress =  9%", 9, true},
		{"unrelated init log", "read_audio_data: reading audio data from 'audio.wav' ...", 0, false},
		{"segment text", "[00:00:00.000 --> 00:00:02.740]   (dramatic music)", 0, false},
		{"output save message", "output_json: saving output to 'result.json'", 0, false},
		{"empty", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseProgressLine(tc.line)
			if ok != tc.ok {
				t.Fatalf("parseProgressLine(%q) ok = %v, want %v", tc.line, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("parseProgressLine(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}
