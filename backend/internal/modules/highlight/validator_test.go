package highlight

import "testing"

func TestStripCodeFence(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain json, no fence", `{"game":"mlbb"}`, `{"game":"mlbb"}`},
		{"fenced with json language tag", "```json\n{\"game\":\"mlbb\"}\n```", `{"game":"mlbb"}`},
		{"fenced without language tag", "```\n{\"game\":\"mlbb\"}\n```", `{"game":"mlbb"}`},
		{"fenced with surrounding whitespace", "  \n```json\n{\"game\":\"mlbb\"}\n```\n  ", `{"game":"mlbb"}`},
		{"plain json with surrounding whitespace", "  {\"game\":\"mlbb\"}  ", `{"game":"mlbb"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := StripCodeFence(tc.in); got != tc.want {
				t.Errorf("StripCodeFence(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Run("valid object", func(t *testing.T) {
		h, err := Parse(`{"game":"mlbb","ringkasan":"x","segmen":[]}`)
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if h.Game != "mlbb" {
			t.Errorf("Game = %q, want mlbb", h.Game)
		}
	})
	t.Run("malformed json", func(t *testing.T) {
		if _, err := Parse(`{"game":`); err == nil {
			t.Fatal("Parse with malformed JSON: want error, got nil")
		}
	})
	t.Run("wrong top-level type", func(t *testing.T) {
		if _, err := Parse(`[1,2,3]`); err == nil {
			t.Fatal("Parse with a JSON array at top level: want error, got nil")
		}
	})
}

func validSegment() Segment {
	return Segment{
		Mulai: "00:01:00", Selesai: "00:02:00", Kategori: "draft",
		Label: "Draft pick kedua tim", Alasan: "caster membahas pick", Narasi: "Tim A mengamankan hero incaran",
	}
}

func validHighlight() *Highlight {
	return &Highlight{
		Game:      "mlbb",
		Ringkasan: "Tim A menang",
		Segmen:    []Segment{validSegment()},
	}
}

var mobaCategories = []string{"draft", "early", "mid", "end", "kesimpulan"}

func TestValidate_FullyValidHasNoErrors(t *testing.T) {
	errs := Validate(validHighlight(), 3600, mobaCategories)
	if len(errs) != 0 {
		t.Errorf("Validate() = %+v, want no errors", errs)
	}
}

func TestValidate_MissingGameAndRingkasan(t *testing.T) {
	h := validHighlight()
	h.Game = ""
	h.Ringkasan = "  "

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 0, "game")
	assertHasFieldError(t, errs, 0, "ringkasan")
}

func TestValidate_EmptySegmenList(t *testing.T) {
	h := validHighlight()
	h.Segmen = nil

	errs := Validate(h, 3600, mobaCategories)
	if len(errs) != 1 || errs[0].Field != "segmen" {
		t.Fatalf("Validate() = %+v, want exactly one 'segmen' error", errs)
	}
}

func TestValidate_InvalidTimeFormat(t *testing.T) {
	cases := []struct {
		name  string
		mulai string
	}{
		{"not HH:MM:SS at all", "1:05"},
		{"minutes out of range", "00:61:00"},
		{"seconds out of range", "00:01:61"},
		{"empty", ""},
		{"with milliseconds", "00:01:05.500"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := validHighlight()
			h.Segmen[0].Mulai = tc.mulai
			errs := Validate(h, 3600, mobaCategories)
			assertHasFieldError(t, errs, 1, "mulai")
		})
	}
}

func TestValidate_MulaiNotBeforeSelesai(t *testing.T) {
	cases := []struct {
		name           string
		mulai, selesai string
	}{
		{"equal", "00:01:00", "00:01:00"},
		{"mulai after selesai", "00:02:00", "00:01:00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := validHighlight()
			h.Segmen[0].Mulai = tc.mulai
			h.Segmen[0].Selesai = tc.selesai
			errs := Validate(h, 3600, mobaCategories)
			assertHasFieldError(t, errs, 1, "mulai")
		})
	}
}

func TestValidate_SelesaiExceedsVideoDuration(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Mulai = "00:59:00"
	h.Segmen[0].Selesai = "01:00:01"

	errs := Validate(h, 3600, mobaCategories) // video is exactly 1 hour (3600s)
	assertHasFieldError(t, errs, 1, "selesai")
}

func TestValidate_SelesaiExactlyAtDurationIsAllowed(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Mulai = "00:59:00"
	h.Segmen[0].Selesai = "01:00:00"

	errs := Validate(h, 3600, mobaCategories)
	if hasFieldError(errs, 1, "selesai") {
		t.Errorf("Validate() = %+v, selesai exactly at durationSec should not be an error", errs)
	}
}

func TestValidate_KategoriMissing(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Kategori = "  "

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 1, "kategori")
}

func TestValidate_KategoriNotInGenreList(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Kategori = "drop" // valid for battle royale, not moba

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 1, "kategori")
}

func TestValidate_KategoriValidForGenre(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Kategori = "kesimpulan"

	errs := Validate(h, 3600, mobaCategories)
	if hasFieldError(errs, 1, "kategori") {
		t.Errorf("Validate() = %+v, 'kesimpulan' is a valid moba category", errs)
	}
}

func TestValidate_LabelMissing(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Label = ""

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 1, "label")
}

func TestValidate_LabelTooLong(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Label = "satu dua tiga empat lima enam tujuh delapan sembilan"

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 1, "label")
}

func TestValidate_LabelExactlyEightWordsIsAllowed(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Label = "satu dua tiga empat lima enam tujuh delapan"

	errs := Validate(h, 3600, mobaCategories)
	if hasFieldError(errs, 1, "label") {
		t.Errorf("Validate() = %+v, an 8-word label should not be an error", errs)
	}
}

func TestValidate_AlasanMissing(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Alasan = ""

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 1, "alasan")
}

func TestValidate_NarasiMissing(t *testing.T) {
	h := validHighlight()
	h.Segmen[0].Narasi = ""

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 1, "narasi")
}

func TestValidate_SegmentsOverlap(t *testing.T) {
	h := validHighlight()
	h.Segmen = []Segment{
		{Mulai: "00:01:00", Selesai: "00:02:00", Kategori: "draft", Label: "A", Alasan: "x", Narasi: "y"},
		{Mulai: "00:01:30", Selesai: "00:03:00", Kategori: "early", Label: "B", Alasan: "x", Narasi: "y"},
	}

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 2, "mulai")
}

func TestValidate_SegmentsOutOfOrder(t *testing.T) {
	h := validHighlight()
	h.Segmen = []Segment{
		{Mulai: "00:05:00", Selesai: "00:06:00", Kategori: "draft", Label: "A", Alasan: "x", Narasi: "y"},
		{Mulai: "00:01:00", Selesai: "00:02:00", Kategori: "early", Label: "B", Alasan: "x", Narasi: "y"},
	}

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 2, "mulai")
}

func TestValidate_SegmentsBackToBackIsAllowed(t *testing.T) {
	h := validHighlight()
	h.Segmen = []Segment{
		{Mulai: "00:01:00", Selesai: "00:02:00", Kategori: "draft", Label: "A", Alasan: "x", Narasi: "y"},
		{Mulai: "00:02:00", Selesai: "00:03:00", Kategori: "early", Label: "B", Alasan: "x", Narasi: "y"},
	}

	errs := Validate(h, 3600, mobaCategories)
	if hasFieldError(errs, 2, "mulai") {
		t.Errorf("Validate() = %+v, back-to-back segments (no gap, no overlap) should not be an error", errs)
	}
}

func TestValidate_OneBadTimestampDoesNotCascade(t *testing.T) {
	// Segment 2's unparsable time must not make segment 3's sequencing
	// check blow up or misreport.
	h := validHighlight()
	h.Segmen = []Segment{
		{Mulai: "00:01:00", Selesai: "00:02:00", Kategori: "draft", Label: "A", Alasan: "x", Narasi: "y"},
		{Mulai: "garbage", Selesai: "00:03:00", Kategori: "early", Label: "B", Alasan: "x", Narasi: "y"},
		{Mulai: "00:04:00", Selesai: "00:05:00", Kategori: "mid", Label: "C", Alasan: "x", Narasi: "y"},
	}

	errs := Validate(h, 3600, mobaCategories)
	assertHasFieldError(t, errs, 2, "mulai") // the format error itself
	if hasFieldError(errs, 3, "mulai") {
		t.Errorf("Validate() = %+v, segment 3's own sequencing should not error because of segment 2's bad timestamp", errs)
	}
}

func TestValidate_ReportsSegmentNumbersForMultipleErrors(t *testing.T) {
	h := validHighlight()
	h.Segmen = []Segment{
		validSegment(),
		{Mulai: "00:03:00", Selesai: "00:02:00", Kategori: "", Label: "", Alasan: "", Narasi: ""}, // segment 2: many problems
	}

	errs := Validate(h, 3600, mobaCategories)
	for _, field := range []string{"mulai", "kategori", "label", "alasan", "narasi"} {
		assertHasFieldError(t, errs, 2, field)
	}
	// nothing should be reported against segment 1, which is valid.
	for _, e := range errs {
		if e.Segmen == 1 {
			t.Errorf("unexpected error against valid segment 1: %+v", e)
		}
	}
}

func TestParseHMS(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want float64
		ok   bool
	}{
		{"zero", "00:00:00", 0, true},
		{"typical", "01:02:05", 3725, true},
		{"minute boundary", "00:59:59", 3599, true},
		{"minutes out of range", "00:60:00", 0, false},
		{"seconds out of range", "00:00:60", 0, false},
		{"too few digits", "1:2:3", 0, false},
		{"empty", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseHMS(tc.in)
			if ok != tc.ok {
				t.Fatalf("parseHMS(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("parseHMS(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func hasFieldError(errs []ValidationError, segmen int, field string) bool {
	for _, e := range errs {
		if e.Segmen == segmen && e.Field == field {
			return true
		}
	}
	return false
}

func assertHasFieldError(t *testing.T, errs []ValidationError, segmen int, field string) {
	t.Helper()
	if !hasFieldError(errs, segmen, field) {
		t.Errorf("Validate() = %+v, want an error for segmen=%d field=%q", errs, segmen, field)
	}
}
