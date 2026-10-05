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
