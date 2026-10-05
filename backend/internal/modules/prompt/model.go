package prompt

import "time"

type GameMode struct {
	Code      string            `json:"code"`
	Name      string            `json:"name"`
	Genre     string            `json:"genre"`
	BlockCode string            `json:"block_code"`
	Terms     map[string]string `json:"terms"`
	SortOrder int               `json:"sort_order"`
}

type Block struct {
	Code       string    `json:"code"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	Categories []string  `json:"categories"`
	IsCustom   bool      `json:"is_custom"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// UpdateBlockRequest edits a block's body and/or its valid category list,
// per docs/prd.md ("Mode baru bisa ditambahkan ... dengan menyalin blok
// tugas yang ada, lalu mengubah kategori dan istilahnya").
type UpdateBlockRequest struct {
	Body       string   `json:"body"`
	Categories []string `json:"categories"`
}

// AssembleInput is the project data needed to fill in the frame's
// placeholders, per docs/prd.md ("Kerangka bersama").
type AssembleInput struct {
	GameCode      string
	VideoTitle    string
	DurationSec   float64
	TeamA         string
	TeamB         string
	TargetMinutes int
}
