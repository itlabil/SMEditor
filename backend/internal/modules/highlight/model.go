package highlight

type Segment struct {
	Mulai    string `json:"mulai"`
	Selesai  string `json:"selesai"`
	Kategori string `json:"kategori"`
	Label    string `json:"label"`
	Alasan   string `json:"alasan"`
	Narasi   string `json:"narasi"`
}

// TeamDraft is one team's draft result. Pick is in pick order.
type TeamDraft struct {
	Nama string   `json:"nama"`
	Pick []string `json:"pick"`
	Ban  []string `json:"ban"`
}

// Draft is the optional MOBA draft result (SM-16). The Premiere plugin
// only reads "segmen", so this field never affects it.
type Draft struct {
	TimA TeamDraft `json:"tim_a"`
	TimB TeamDraft `json:"tim_b"`
}

// Highlight is the JSON shape requested from the AI, per docs/prd.md.
type Highlight struct {
	Game      string    `json:"game"`
	Ringkasan string    `json:"ringkasan"`
	Draft     *Draft    `json:"draft,omitempty"`
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
	Draft     *Draft    `json:"draft,omitempty"`
	Segmen    []Segment `json:"segmen"`
	Video     string    `json:"video"`
	Durasi    float64   `json:"durasi"`
}

// toHighlight drops the app-added fields so a saved highlight can be
// edited and re-checked with Validate. The segment slice is copied so
// edits never alias the caller's SavedHighlight.
func (s *SavedHighlight) toHighlight() *Highlight {
	segmen := make([]Segment, len(s.Segmen))
	copy(segmen, s.Segmen)
	return &Highlight{Game: s.Game, Ringkasan: s.Ringkasan, Draft: s.Draft, Segmen: segmen}
}

// SegmentInput is the edit form for one segment on the project page
// (SM-15). "alasan" is not editable there and is kept as-is.
type SegmentInput struct {
	Mulai    string `json:"mulai"`
	Selesai  string `json:"selesai"`
	Kategori string `json:"kategori"`
	Label    string `json:"label"`
	Narasi   string `json:"narasi"`
}
