package whisper

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Segment is one transcribed line, with time in seconds.
type Segment struct {
	StartSec float64
	EndSec   float64
	Text     string
}

// Transcript is whisper-cli's output converted to SMEditor's shape, per
// docs/prd.md ("Format data").
type Transcript struct {
	Language    string
	DurationSec float64
	Segments    []Segment
}

type rawOutput struct {
	Result struct {
		Language string `json:"language"`
	} `json:"result"`
	Transcription []struct {
		Offsets struct {
			From int64 `json:"from"`
			To   int64 `json:"to"`
		} `json:"offsets"`
		Text string `json:"text"`
	} `json:"transcription"`
}

// ParseRawOutput converts whisper-cli's own JSON output (written with -oj)
// into SMEditor's transcript shape. Segment text is trimmed; empty
// segments and immediate consecutive duplicates (whisper.cpp sometimes
// repeats a line verbatim when it loses track) are dropped, per
// .agents/skills/sm-external-tools. durationSec comes from the project's
// already-probed video metadata, since whisper's own output has no
// single "total duration" field.
func ParseRawOutput(data []byte, durationSec float64) (Transcript, error) {
	var raw rawOutput
	if err := json.Unmarshal(data, &raw); err != nil {
		return Transcript{}, fmt.Errorf("baca keluaran whisper: %w", err)
	}

	t := Transcript{Language: raw.Result.Language, DurationSec: durationSec}
	var lastText string
	for _, seg := range raw.Transcription {
		text := strings.TrimSpace(seg.Text)
		if text == "" || text == lastText {
			continue
		}
		lastText = text
		t.Segments = append(t.Segments, Segment{
			StartSec: float64(seg.Offsets.From) / 1000,
			EndSec:   float64(seg.Offsets.To) / 1000,
			Text:     text,
		})
	}
	return t, nil
}

type jsonSegment struct {
	Mulai   float64 `json:"mulai"`
	Selesai float64 `json:"selesai"`
	Teks    string  `json:"teks"`
}

type jsonTranscript struct {
	Bahasa string        `json:"bahasa"`
	Durasi float64       `json:"durasi"`
	Segmen []jsonSegment `json:"segmen"`
}

// JSON renders the full transcript.json content, per docs/prd.md.
func (t Transcript) JSON() ([]byte, error) {
	out := jsonTranscript{Bahasa: t.Language, Durasi: t.DurationSec, Segmen: []jsonSegment{}}
	for _, s := range t.Segments {
		out.Segmen = append(out.Segmen, jsonSegment{Mulai: s.StartSec, Selesai: s.EndSec, Teks: s.Text})
	}
	return json.MarshalIndent(out, "", "  ")
}

// TXT renders the condensed transcript.txt content ("[HH:MM:SS] text" per
// line), the format sent to the AI, per docs/prd.md.
func (t Transcript) TXT() string {
	var b strings.Builder
	for _, s := range t.Segments {
		b.WriteString("[")
		b.WriteString(formatHMS(s.StartSec))
		b.WriteString("] ")
		b.WriteString(s.Text)
		b.WriteString("\n")
	}
	return b.String()
}

func formatHMS(sec float64) string {
	total := int(sec)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
