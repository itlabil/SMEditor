package ffmpeg

import (
	"os"
	"path/filepath"
	"testing"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	return data
}

func TestParseProbeOutput_H264Aac(t *testing.T) {
	data := readTestdata(t, "probe_h264_aac.json")

	m, err := parseProbeOutput(data)
	if err != nil {
		t.Fatalf("parseProbeOutput: %v", err)
	}

	if m.VideoCodec != "h264" {
		t.Errorf("VideoCodec = %q, want h264", m.VideoCodec)
	}
	if m.Width != 1920 || m.Height != 1080 {
		t.Errorf("Width/Height = %d/%d, want 1920/1080", m.Width, m.Height)
	}
	if m.FPS != 24 {
		t.Errorf("FPS = %v, want 24", m.FPS)
	}
	if m.DurationSec != 150.140227 {
		t.Errorf("DurationSec = %v, want 150.140227", m.DurationSec)
	}
	if m.SizeBytes != 40044118 {
		t.Errorf("SizeBytes = %d, want 40044118", m.SizeBytes)
	}
}

func TestParseProbeOutput_VP9NeedsConversion(t *testing.T) {
	data := readTestdata(t, "probe_vp9_opus.json")

	m, err := parseProbeOutput(data)
	if err != nil {
		t.Fatalf("parseProbeOutput: %v", err)
	}

	if m.VideoCodec != "vp9" {
		t.Errorf("VideoCodec = %q, want vp9", m.VideoCodec)
	}
	if m.Width != 3840 || m.Height != 2160 {
		t.Errorf("Width/Height = %d/%d, want 3840/2160", m.Width, m.Height)
	}
	wantFPS := 30000.0 / 1001.0
	if m.FPS != wantFPS {
		t.Errorf("FPS = %v, want %v", m.FPS, wantFPS)
	}
}

func TestParseProbeOutput_InvalidJSON(t *testing.T) {
	if _, err := parseProbeOutput([]byte("not json")); err == nil {
		t.Fatal("parseProbeOutput with invalid JSON: want error, got nil")
	}
}

func TestParseFrameRate(t *testing.T) {
	cases := []struct {
		name string
		rate string
		want float64
	}{
		{"whole number", "24/1", 24},
		{"ntsc fraction", "30000/1001", 30000.0 / 1001.0},
		{"ntsc 60", "60000/1001", 60000.0 / 1001.0},
		{"unknown (audio stream)", "0/0", 0},
		{"malformed, no slash", "abc", 0},
		{"malformed numerator", "abc/1", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseFrameRate(tc.rate); got != tc.want {
				t.Errorf("parseFrameRate(%q) = %v, want %v", tc.rate, got, tc.want)
			}
		})
	}
}
