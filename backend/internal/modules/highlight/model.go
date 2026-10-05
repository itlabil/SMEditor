package highlight

type Segment struct {
	Mulai    string `json:"mulai"`
	Selesai  string `json:"selesai"`
	Kategori string `json:"kategori"`
	Label    string `json:"label"`
	Alasan   string `json:"alasan"`
	Narasi   string `json:"narasi"`
}

// Highlight is the JSON shape requested from the AI, per docs/prd.md.
type Highlight struct {
	Game      string    `json:"game"`
	Ringkasan string    `json:"ringkasan"`
	Segmen    []Segment `json:"segmen"`
}

// ValidationError is one problem found while checking a Highlight, per
// docs/flow.md section 4.5. Segmen is 1-indexed to match what the user
// sees on screen; 0 means the problem is with the document as a whole,
// not any particular segment.
type ValidationError struct {
	Segmen  int    `json:"segmen"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// SavedHighlight is what gets written to highlight.json: the AI's JSON
// plus the two fields the app adds so the Premiere plugin can match the
// file and check time bounds, per docs/prd.md.
type SavedHighlight struct {
	Game      string    `json:"game"`
	Ringkasan string    `json:"ringkasan"`
	Segmen    []Segment `json:"segmen"`
	Video     string    `json:"video"`
	Durasi    float64   `json:"durasi"`
}
