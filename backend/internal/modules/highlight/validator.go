package highlight

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var codeFencePattern = regexp.MustCompile("(?s)```(?:[a-zA-Z0-9]*\\n)?(.*?)```")

// StripCodeFence removes a surrounding ```json ... ``` (or plain ``` ...
// ```) block if the AI wrapped its answer in one, per docs/flow.md
// section 4.5 ("App membuang pembungkus blok kode jika ada"). Plain JSON
// with no fence is returned unchanged.
func StripCodeFence(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if m := codeFencePattern.FindStringSubmatch(trimmed); m != nil {
		return strings.TrimSpace(m[1])
	}
	return trimmed
}

// Parse decodes cleaned (already fence-stripped) JSON. It only checks
// that the JSON itself is well-formed; field- and segment-level rules
// are checked separately by Validate.
func Parse(cleaned string) (*Highlight, error) {
	var h Highlight
	if err := json.Unmarshal([]byte(cleaned), &h); err != nil {
		return nil, fmt.Errorf("JSON tidak bisa dibaca: %w", err)
	}
	return &h, nil
}

var timePattern = regexp.MustCompile(`^(\d{2}):(\d{2}):(\d{2})$`)

const maxLabelWords = 8

// Validate checks every rule from docs/flow.md section 4.5 (plus the
// field formats from docs/prd.md) and returns one ValidationError per
// problem found. A nil/empty result means h is safe to save.
func Validate(h *Highlight, durationSec float64, validCategories []string) []ValidationError {
	var errs []ValidationError

	if strings.TrimSpace(h.Game) == "" {
		errs = append(errs, ValidationError{Field: "game", Message: "Field game wajib diisi"})
	}
	if strings.TrimSpace(h.Ringkasan) == "" {
		errs = append(errs, ValidationError{Field: "ringkasan", Message: "Field ringkasan wajib diisi"})
	}
	errs = append(errs, validateDraft(h.Draft)...)
	if len(h.Segmen) == 0 {
		errs = append(errs, ValidationError{Field: "segmen", Message: "Tidak ada segmen"})
		return errs
	}

	validCategorySet := make(map[string]bool, len(validCategories))
	for _, c := range validCategories {
		validCategorySet[c] = true
	}

	starts := make([]float64, len(h.Segmen))
	ends := make([]float64, len(h.Segmen))
	timesOK := make([]bool, len(h.Segmen))

	for i, seg := range h.Segmen {
		n := i + 1

		mulaiSec, mulaiOK := parseHMS(seg.Mulai)
		if !mulaiOK {
			errs = append(errs, ValidationError{Segmen: n, Field: "mulai", Message: "Format waktu harus HH:MM:SS"})
		}
		selesaiSec, selesaiOK := parseHMS(seg.Selesai)
		if !selesaiOK {
			errs = append(errs, ValidationError{Segmen: n, Field: "selesai", Message: "Format waktu harus HH:MM:SS"})
		}
		if mulaiOK && selesaiOK {
			if mulaiSec >= selesaiSec {
				errs = append(errs, ValidationError{Segmen: n, Field: "mulai", Message: "Waktu mulai harus lebih kecil dari selesai"})
			}
			if selesaiSec > durationSec {
				errs = append(errs, ValidationError{Segmen: n, Field: "selesai", Message: "Melewati durasi video"})
			}
		}
		starts[i], ends[i], timesOK[i] = mulaiSec, selesaiSec, mulaiOK && selesaiOK

		if strings.TrimSpace(seg.Kategori) == "" {
			errs = append(errs, ValidationError{Segmen: n, Field: "kategori", Message: "Kategori wajib diisi"})
		} else if len(validCategorySet) > 0 && !validCategorySet[seg.Kategori] {
			errs = append(errs, ValidationError{Segmen: n, Field: "kategori", Message: "Kategori tidak dikenal untuk genre ini (" + strings.Join(validCategories, ", ") + ")"})
		}

		label := strings.TrimSpace(seg.Label)
		if label == "" {
			errs = append(errs, ValidationError{Segmen: n, Field: "label", Message: "Label wajib diisi"})
		} else if words := len(strings.Fields(label)); words > maxLabelWords {
			errs = append(errs, ValidationError{Segmen: n, Field: "label", Message: fmt.Sprintf("Label maksimal %d kata", maxLabelWords)})
		}

		if strings.TrimSpace(seg.Alasan) == "" {
			errs = append(errs, ValidationError{Segmen: n, Field: "alasan", Message: "Alasan wajib diisi"})
		}
		if strings.TrimSpace(seg.Narasi) == "" {
			errs = append(errs, ValidationError{Segmen: n, Field: "narasi", Message: "Narasi wajib diisi"})
		}
	}

	// Sequential and non-overlapping: each segment's start must be at or
	// after the previous one's end. Only checked when both segments'
	// times parsed, so one bad timestamp doesn't cascade into every
	// segment after it.
	for i := 1; i < len(h.Segmen); i++ {
		if !timesOK[i] || !timesOK[i-1] {
			continue
		}
		if starts[i] < ends[i-1] {
			errs = append(errs, ValidationError{Segmen: i + 1, Field: "mulai", Message: "Tumpang tindih atau tidak berurutan dengan segmen sebelumnya"})
		}
	}

	return errs
}

const maxDraftPicks = 5

// validateDraft checks the optional "draft" field (SM-16): absent is
// fine; if present, each team may have at most 5 picks, and an empty
// ban list is allowed. Errors are document-level (Segmen 0).
func validateDraft(d *Draft) []ValidationError {
	if d == nil {
		return nil
	}
	var errs []ValidationError
	for _, team := range []struct {
		key  string
		data TeamDraft
	}{{"tim_a", d.TimA}, {"tim_b", d.TimB}} {
		if len(team.data.Pick) > maxDraftPicks {
			errs = append(errs, ValidationError{
				Field:   "draft." + team.key + ".pick",
				Message: fmt.Sprintf("Pick maksimal %d hero, ada %d", maxDraftPicks, len(team.data.Pick)),
			})
		}
	}
	return errs
}

const (
	minSegmentDurationSec = 15.0
	maxSegmentDurationSec = 150.0
)

// Warnings flags segments whose duration is unusually short (<15s) or
// long (>150s). Unlike Validate, these never block saving - Save calls
// this only after Validate has already passed, purely so the segment
// list can flag them for the user to double-check manually.
func Warnings(h *Highlight) []ValidationError {
	warnings := []ValidationError{}
	for i, seg := range h.Segmen {
		start, startOK := parseHMS(seg.Mulai)
		end, endOK := parseHMS(seg.Selesai)
		if !startOK || !endOK {
			continue // Validate already flags a bad format; don't double up
		}

		dur := end - start
		switch {
		case dur < minSegmentDurationSec:
			warnings = append(warnings, ValidationError{
				Segmen:  i + 1,
				Field:   "durasi",
				Message: fmt.Sprintf("Durasi %.0f detik, lebih pendek dari %g detik", dur, minSegmentDurationSec),
			})
		case dur > maxSegmentDurationSec:
			warnings = append(warnings, ValidationError{
				Segmen:  i + 1,
				Field:   "durasi",
				Message: fmt.Sprintf("Durasi %.0f detik, lebih panjang dari %g detik", dur, maxSegmentDurationSec),
			})
		}
	}
	return warnings
}

func parseHMS(s string) (float64, bool) {
	m := timePattern.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	se, _ := strconv.Atoi(m[3])
	if mi >= 60 || se >= 60 {
		return 0, false
	}
	return float64(h*3600 + mi*60 + se), true
}
