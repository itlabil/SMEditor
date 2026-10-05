package project

import "time"

// Status values, per docs/flow.md section 3.
const (
	StatusBaru              = "baru"
	StatusMengunduh         = "mengunduh"
	StatusGagalUnduh        = "gagal_unduh"
	StatusTranscript        = "transcript"
	StatusGagalTranscript   = "gagal_transcript"
	StatusMenungguHighlight = "menunggu_highlight"
	StatusSiapPremiere      = "siap_premiere"
)

type Project struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	GameCode       string    `json:"game_code"`
	YoutubeURL     string    `json:"youtube_url"`
	VideoTitle     string    `json:"video_title"`
	TeamA          string    `json:"team_a"`
	TeamB          string    `json:"team_b"`
	TargetMinutes  int       `json:"target_minutes"`
	Status         string    `json:"status"`
	ErrorMessage   string    `json:"error_message"`
	DurationSec    float64   `json:"duration_sec"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	FPS            float64   `json:"fps"`
	VideoCodec     string    `json:"video_codec"`
	VideoFile      string    `json:"video_file"`
	TranscriptLang string    `json:"transcript_lang"`
	HasHighlight   bool      `json:"has_highlight"`
	SizeBytes      int64     `json:"size_bytes"`
	HasThumbnail   bool      `json:"has_thumbnail"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name          string `json:"name"`
	GameCode      string `json:"game_code"`
	YoutubeURL    string `json:"youtube_url"`
	TeamA         string `json:"team_a"`
	TeamB         string `json:"team_b"`
	TargetMinutes int    `json:"target_minutes"`
}

// ExportRequest is the body of POST /api/projects/:id/export ("Salin ke
// folder"). Overwrite must be true to proceed when TargetDir (see
// ExportResult) already exists; otherwise the request fails with
// export_dir_exists so the UI can ask for confirmation first.
type ExportRequest struct {
	DestDir   string `json:"dest_dir"`
	Overwrite bool   `json:"overwrite"`
}

// ExportResult is what POST /api/projects/:id/export returns once the
// export job is queued: the job to track over SSE, and the absolute
// folder (DestDir plus the project's sanitized name) the copy will land
// in, so the UI can show it once the job finishes.
type ExportResult struct {
	JobID     string      `json:"job_id"`
	TargetDir string      `json:"target_dir"`
	Files     ExportFiles `json:"files"`
}

// ExportFiles are the file names "Salin ke folder" writes inside
// TargetDir (SM-17): named after the project so several games can sit in
// one Premiere project folder without clashing. Files inside
// data/projects/<id>/ keep their fixed names.
type ExportFiles struct {
	Video     string `json:"video"`
	Highlight string `json:"highlight"`
	Narasi    string `json:"narasi"`
}
