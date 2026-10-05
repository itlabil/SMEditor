package settings

// Keys match the settings table rows documented in docs/erd.md.
const (
	KeyYtdlpPath     = "ytdlp_path"
	KeyFfmpegPath    = "ffmpeg_path"
	KeyFfprobePath   = "ffprobe_path"
	KeyWhisperPath   = "whisper_path"
	KeyWhisperModel  = "whisper_model"
	KeyWhisperDevice = "whisper_device"
	// KeyExportDir is the last destination folder typed into "Salin ke
	// folder" (project.Service.ExportToFolder), remembered as the next
	// default, per docs/prd.md.
	KeyExportDir = "export_dir"
)

// defaults are used for any key not yet saved in the settings table.
// ffmpeg and ffprobe default to bare names resolved from the system PATH;
// yt-dlp and whisper ship as bundled binaries under tools/. export_dir
// has no sensible default location, so it starts empty until the user
// exports once.
var defaults = map[string]string{
	KeyYtdlpPath:     "tools/yt-dlp",
	KeyFfmpegPath:    "ffmpeg",
	KeyFfprobePath:   "ffprobe",
	KeyWhisperPath:   "tools/whisper-cli",
	KeyWhisperModel:  "tools/models/ggml-medium.bin",
	KeyWhisperDevice: "auto",
	KeyExportDir:     "",
}

var validWhisperDevices = map[string]bool{"auto": true, "cpu": true, "gpu": true}

// CheckResult reports one tool's check outcome. Path is the final resolved
// path that was actually used (after PATH lookup or BaseDir join), not the
// raw settings value.
type CheckResult struct {
	Tool    string `json:"tool"`
	Path    string `json:"path"`
	Found   bool   `json:"found"`
	Version string `json:"version"`
	// Backend is only set for whisper: "gpu" or "cpu", read from the
	// backend lines whisper-cli prints at startup, or "" when its output
	// does not say. GPU is the CUDA device name when one was found.
	Backend string `json:"backend,omitempty"`
	GPU     string `json:"gpu,omitempty"`
}
