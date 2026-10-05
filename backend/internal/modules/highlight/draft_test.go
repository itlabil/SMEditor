package highlight

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"smeditor/internal/httpx"
	"smeditor/internal/storage"
)

func heroes(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "Hero" + string(rune('A'+i))
	}
	return out
}

func TestValidate_Draft(t *testing.T) {
	cases := []struct {
		name      string
		draft     *Draft
		wantField string // "" means no draft error expected
	}{
		{"absent", nil, ""},
		{"five picks each, empty bans", &Draft{TimA: TeamDraft{Nama: "ONIC", Pick: heroes(5)}, TimB: TeamDraft{Nama: "RRQ", Pick: heroes(5), Ban: []string{}}}, ""},
		{"partial picks", &Draft{TimA: TeamDraft{Pick: heroes(2)}, TimB: TeamDraft{}}, ""},
		{"six picks tim_a", &Draft{TimA: TeamDraft{Pick: heroes(6)}, TimB: TeamDraft{Pick: heroes(5)}}, "draft.tim_a.pick"},
		{"six picks tim_b", &Draft{TimA: TeamDraft{Pick: heroes(5)}, TimB: TeamDraft{Pick: heroes(6)}}, "draft.tim_b.pick"},
		{"many bans allowed", &Draft{TimA: TeamDraft{Pick: heroes(5), Ban: heroes(5)}, TimB: TeamDraft{Pick: heroes(5), Ban: heroes(5)}}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := validHighlight()
			h.Draft = tc.draft
			errs := Validate(h, 3600, mobaCategories)
			if tc.wantField == "" {
				if len(errs) != 0 {
					t.Errorf("Validate() = %+v, want no errors", errs)
				}
				return
			}
			assertHasFieldError(t, errs, 0, tc.wantField)
		})
	}
}

func TestParse_Draft(t *testing.T) {
	h, err := Parse(`{"game":"mlbb","ringkasan":"x","draft":{"tim_a":{"nama":"ONIC","pick":["Fanny","Kaja"],"ban":["Ling"]},"tim_b":{"nama":"RRQ","pick":["Lancelot"],"ban":[]}},"segmen":[]}`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := &Draft{
		TimA: TeamDraft{Nama: "ONIC", Pick: []string{"Fanny", "Kaja"}, Ban: []string{"Ling"}},
		TimB: TeamDraft{Nama: "RRQ", Pick: []string{"Lancelot"}, Ban: []string{}},
	}
	if !reflect.DeepEqual(h.Draft, want) {
		t.Errorf("Draft = %+v, want %+v", h.Draft, want)
	}
}

func TestNormalizeDraft(t *testing.T) {
	got := normalizeDraft(&Draft{
		TimA: TeamDraft{Nama: " ONIC ", Pick: []string{" Fanny", "", "  "}, Ban: nil},
		TimB: TeamDraft{},
	})
	want := &Draft{
		TimA: TeamDraft{Nama: "ONIC", Pick: []string{"Fanny"}, Ban: []string{}},
		TimB: TeamDraft{Nama: "", Pick: []string{}, Ban: []string{}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeDraft() = %+v, want %+v", got, want)
	}
	if normalizeDraft(nil) != nil {
		t.Error("normalizeDraft(nil) should stay nil")
	}
}

func TestBuildNarasi_DraftFirst(t *testing.T) {
	draft := &Draft{
		TimA: TeamDraft{Nama: "ONIC", Pick: []string{"Fanny", "Kaja"}, Ban: []string{"Ling"}},
		TimB: TeamDraft{Pick: []string{"Lancelot"}, Ban: []string{}},
	}
	segmen := []Segment{{Mulai: "00:01:00", Selesai: "00:02:00", Label: "Draft", Narasi: "Narasi"}}
	want := "HASIL DRAFT\n" +
		"ONIC\n  Pick: Fanny, Kaja\n  Ban: Ling\n" +
		"Tim B\n  Pick: Lancelot\n  Ban: -\n" +
		"\n" +
		"[00:01:00 - 00:02:00] Draft\nNarasi\n\n"
	if got := buildNarasi(draft, segmen); got != want {
		t.Errorf("buildNarasi() = %q, want %q", got, want)
	}
}

const draftHighlightJSON = `{
  "game": "mlbb",
  "ringkasan": "Tim A menang",
  "draft": {
    "tim_a": {"nama": "ONIC", "pick": ["Fanny", "Kaja", "Valentina", "Claude", "Grock"], "ban": ["Ling"]},
    "tim_b": {"nama": "RRQ", "pick": ["Lancelot"], "ban": []}
  },
  "segmen": [
    {"mulai":"00:01:00","selesai":"00:02:00","kategori":"draft","label":"Draft pick","alasan":"caster membahas pick","narasi":"Narasi draft"},
    {"mulai":"00:05:00","selesai":"00:05:40","kategori":"early","label":"First blood","alasan":"first blood","narasi":"Narasi early"}
  ]
}`

func TestServiceSave_WithDraft(t *testing.T) {
	svc, st, _ := newTestService(t)
	if _, err := svc.Save(context.Background(), testProjectID, draftHighlightJSON); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var onDisk map[string]any
	if err := json.Unmarshal([]byte(readProjectFile(t, st, storage.HighlightFile)), &onDisk); err != nil {
		t.Fatalf("highlight.json: %v", err)
	}
	if _, ok := onDisk["draft"]; !ok {
		t.Error("highlight.json has no draft field")
	}
	if segs, _ := onDisk["segmen"].([]any); len(segs) != 2 {
		t.Errorf("highlight.json segmen = %v, want 2 segments next to draft", onDisk["segmen"])
	}
	if !strings.HasPrefix(readProjectFile(t, st, storage.NarrationFile), "HASIL DRAFT\nONIC\n  Pick: Fanny, Kaja, Valentina, Claude, Grock\n") {
		t.Errorf("narasi.txt does not start with the draft:\n%s", readProjectFile(t, st, storage.NarrationFile))
	}
}

func TestServiceSave_WithoutDraftOmitsField(t *testing.T) {
	svc, st, _ := newTestService(t)
	if _, err := svc.Save(context.Background(), testProjectID, validHighlightJSON); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var onDisk map[string]any
	if err := json.Unmarshal([]byte(readProjectFile(t, st, storage.HighlightFile)), &onDisk); err != nil {
		t.Fatalf("highlight.json: %v", err)
	}
	if _, ok := onDisk["draft"]; ok {
		t.Error("highlight.json should not contain a draft field when the AI sent none")
	}
	if strings.Contains(readProjectFile(t, st, storage.NarrationFile), "HASIL DRAFT") {
		t.Error("narasi.txt should not contain a draft section")
	}
}

func TestServiceSave_TooManyPicksWritesNothing(t *testing.T) {
	svc, st, _ := newTestService(t)
	body := strings.Replace(draftHighlightJSON, `"pick": ["Lancelot"]`, `"pick": ["A","B","C","D","E","F"]`, 1)
	_, err := svc.Save(context.Background(), testProjectID, body)
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Code != "highlight_invalid" {
		t.Fatalf("err = %v, want highlight_invalid", err)
	}
	assertNoHighlightFilesWritten(t, st)
}

func TestServiceUpdateDraft(t *testing.T) {
	svc, st, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Save(ctx, testProjectID, validHighlightJSON); err != nil {
		t.Fatalf("Save: %v", err)
	}

	saved, err := svc.UpdateDraft(ctx, testProjectID, Draft{
		TimA: TeamDraft{Nama: "ONIC", Pick: []string{"Fanny"}},
		TimB: TeamDraft{Nama: "RRQ", Pick: []string{"Lancelot"}, Ban: []string{"Ling"}},
	})
	if err != nil {
		t.Fatalf("UpdateDraft: %v", err)
	}
	if saved.Draft == nil || saved.Draft.TimB.Ban[0] != "Ling" || len(saved.Segmen) != 1 {
		t.Errorf("saved = %+v", saved)
	}
	if !strings.Contains(readProjectFile(t, st, storage.NarrationFile), "RRQ\n  Pick: Lancelot\n  Ban: Ling\n") {
		t.Error("narasi.txt not rewritten with the new draft")
	}

	before := readProjectFile(t, st, storage.HighlightFile)
	_, err = svc.UpdateDraft(ctx, testProjectID, Draft{TimA: TeamDraft{Pick: heroes(6)}})
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Code != "highlight_invalid" {
		t.Fatalf("err = %v, want highlight_invalid", err)
	}
	if readProjectFile(t, st, storage.HighlightFile) != before {
		t.Error("highlight.json changed despite a validation error")
	}
}

func TestServiceUpdateSegment_KeepsDraft(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Save(ctx, testProjectID, draftHighlightJSON); err != nil {
		t.Fatalf("Save: %v", err)
	}
	saved, err := svc.UpdateSegment(ctx, testProjectID, 2, validEdit())
	if err != nil {
		t.Fatalf("UpdateSegment: %v", err)
	}
	if saved.Draft == nil || saved.Draft.TimA.Nama != "ONIC" {
		t.Errorf("draft lost after editing a segment: %+v", saved.Draft)
	}
}
