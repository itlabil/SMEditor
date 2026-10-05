package whisper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestParseRawOutput_RealWhisperOutput(t *testing.T) {
	data := readTestdata(t, "whisper_output.json")

	tr, err := ParseRawOutput(data, 150.14)
	if err != nil {
		t.Fatalf("ParseRawOutput: %v", err)
	}

	if tr.Language != "en" {
		t.Errorf("Language = %q, want en", tr.Language)
	}
	if tr.DurationSec != 150.14 {
		t.Errorf("DurationSec = %v, want 150.14", tr.DurationSec)
	}
	// The real fixture has 6 raw segments, 5 of which say "(dramatic
	// music)" back to back; consecutive duplicates collapse to 1, so the
	// final count is 2: "(dramatic music)" once, then "(upbeat music)".
	if len(tr.Segments) != 2 {
		t.Fatalf("len(Segments) = %d, want 2 (consecutive duplicates collapsed): %+v", len(tr.Segments), tr.Segments)
	}
	if tr.Segments[0].Text != "(dramatic music)" {
		t.Errorf("Segments[0].Text = %q, want %q", tr.Segments[0].Text, "(dramatic music)")
	}
	if tr.Segments[0].StartSec != 0 || tr.Segments[0].EndSec != 2.74 {
		t.Errorf("Segments[0] times = %v/%v, want 0/2.74", tr.Segments[0].StartSec, tr.Segments[0].EndSec)
	}
	if tr.Segments[1].Text != "(upbeat music)" {
		t.Errorf("Segments[1].Text = %q, want %q", tr.Segments[1].Text, "(upbeat music)")
	}
	if tr.Segments[1].StartSec != 150 {
		t.Errorf("Segments[1].StartSec = %v, want 150", tr.Segments[1].StartSec)
	}
}

func TestParseRawOutput_DropsEmptyAndCollapsesDuplicates(t *testing.T) {
	data := readTestdata(t, "whisper_output_dupes.json")

	tr, err := ParseRawOutput(data, 20)
	if err != nil {
		t.Fatalf("ParseRawOutput: %v", err)
	}

	want := []string{"Halo semua", "selamat datang di pertandingan", "babak pertama dimulai"}
	if len(tr.Segments) != len(want) {
		t.Fatalf("len(Segments) = %d, want %d: %+v", len(tr.Segments), len(want), tr.Segments)
	}
	for i, w := range want {
		if tr.Segments[i].Text != w {
			t.Errorf("Segments[%d].Text = %q, want %q", i, tr.Segments[i].Text, w)
		}
	}
}

func TestParseRawOutput_InvalidJSON(t *testing.T) {
	if _, err := ParseRawOutput([]byte("not json"), 100); err == nil {
		t.Fatal("ParseRawOutput with invalid JSON: want error, got nil")
	}
}

func TestTranscriptJSON(t *testing.T) {
	tr := Transcript{
		Language:    "id",
		DurationSec: 90.5,
		Segments: []Segment{
			{StartSec: 12, EndSec: 16.8, Text: "Welcome back"},
		},
	}
	data, err := tr.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got["bahasa"] != "id" {
		t.Errorf("bahasa = %v, want id", got["bahasa"])
	}
	if got["durasi"] != 90.5 {
		t.Errorf("durasi = %v, want 90.5", got["durasi"])
	}
	segmen, ok := got["segmen"].([]any)
	if !ok || len(segmen) != 1 {
		t.Fatalf("segmen = %v, want one entry", got["segmen"])
	}
}

func TestTranscriptJSON_EmptySegmentsIsArrayNotNull(t *testing.T) {
	tr := Transcript{Language: "auto", DurationSec: 10}
	data, err := tr.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(string(data), `"segmen": []`) {
		t.Errorf("JSON() = %s, want segmen to be [] not null", data)
	}
}

func TestTranscriptTXT(t *testing.T) {
	tr := Transcript{
		Segments: []Segment{
			{StartSec: 12, EndSec: 16.8, Text: "Welcome back to game three"},
			{StartSec: 3665, EndSec: 3670, Text: "and we are heading into the draft phase"},
		},
	}
	got := tr.TXT()
	want := "[00:00:12] Welcome back to game three\n[01:01:05] and we are heading into the draft phase\n"
	if got != want {
		t.Errorf("TXT() = %q, want %q", got, want)
	}
}

func TestFormatHMS(t *testing.T) {
	cases := []struct {
		sec  float64
		want string
	}{
		{0, "00:00:00"},
		{12, "00:00:12"},
		{65, "00:01:05"},
		{3665, "01:01:05"},
	}
	for _, tc := range cases {
		if got := formatHMS(tc.sec); got != tc.want {
			t.Errorf("formatHMS(%v) = %q, want %q", tc.sec, got, tc.want)
		}
	}
}
