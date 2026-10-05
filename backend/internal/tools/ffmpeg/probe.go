package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"smeditor/internal/tools"
)

// Metadata is what SMEditor stores about a project's video, per
// docs/erd.md.
type Metadata struct {
	DurationSec float64
	SizeBytes   int64
	Width       int
	Height      int
	FPS         float64
	VideoCodec  string
}

type probeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
	} `json:"streams"`
}

// Probe reads a video file's metadata with ffprobe's JSON output, per
// .agents/skills/sm-external-tools ("Minta keluaran JSON ... Jangan
// mem-parse teks biasa").
func (Ffprobe) Probe(ctx context.Context, ffprobePath, videoPath string) (Metadata, error) {
	resolved, found := tools.ResolveExecutable(ffprobePath)
	if !found {
		return Metadata{}, fmt.Errorf("ffprobe tidak ditemukan di %s", ffprobePath)
	}

	out, err := exec.CommandContext(ctx, resolved, "-v", "error", "-of", "json", "-show_format", "-show_streams", videoPath).Output()
	if err != nil {
		if ctx.Err() != nil {
			return Metadata{}, ctx.Err()
		}
		return Metadata{}, fmt.Errorf("ffprobe %s: %w", videoPath, err)
	}

	parsed, err := parseProbeOutput(out)
	if err != nil {
		return Metadata{}, fmt.Errorf("baca keluaran ffprobe: %w", err)
	}
	return parsed, nil
}

func parseProbeOutput(data []byte) (Metadata, error) {
	var raw probeOutput
	if err := json.Unmarshal(data, &raw); err != nil {
		return Metadata{}, err
	}

	m := Metadata{}
	m.DurationSec, _ = strconv.ParseFloat(raw.Format.Duration, 64)
	m.SizeBytes, _ = strconv.ParseInt(raw.Format.Size, 10, 64)

	for _, s := range raw.Streams {
		if s.CodecType == "video" {
			m.Width = s.Width
			m.Height = s.Height
			m.VideoCodec = s.CodecName
			m.FPS = parseFrameRate(s.RFrameRate)
			break
		}
	}
	return m, nil
}

// parseFrameRate converts ffprobe's fractional fps ("30000/1001") to a
// decimal number.
func parseFrameRate(s string) float64 {
	num, den, ok := strings.Cut(s, "/")
	if !ok {
		return 0
	}
	n, err1 := strconv.ParseFloat(num, 64)
	d, err2 := strconv.ParseFloat(den, 64)
	if err1 != nil || err2 != nil || d == 0 {
		return 0
	}
	return n / d
}
