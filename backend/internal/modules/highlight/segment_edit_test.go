package highlight

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"smeditor/internal/httpx"
	"smeditor/internal/storage"
)

const twoSegmentJSON = `{
  "game": "mlbb",
  "ringkasan": "Tim A menang",
  "segmen": [
    {"mulai":"00:01:00","selesai":"00:02:00","kategori":"draft","label":"Draft pick","alasan":"caster membahas pick","narasi":"Narasi draft"},
    {"mulai":"00:05:00","selesai":"00:05:40","kategori":"early","label":"First blood","alasan":"first blood","narasi":"Narasi early"}
  ]
}`

func readProjectFile(t *testing.T, st *storage.Storage, name string) string {
	t.Helper()
	path, _ := st.FilePath(testProjectID, name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func newEditableService(t *testing.T) (*Service, *storage.Storage) {
	t.Helper()
	svc, st, _ := newTestService(t)
	if _, err := svc.Save(context.Background(), testProjectID, twoSegmentJSON); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return svc, st
}

func validEdit() SegmentInput {
	return SegmentInput{Mulai: "00:05:10", Selesai: "00:05:50", Kategori: "mid", Label: "Team fight", Narasi: "Narasi baru"}
}

func TestServiceUpdateSegment_RewritesFiles(t *testing.T) {
	svc, st := newEditableService(t)

	saved, err := svc.UpdateSegment(context.Background(), testProjectID, 2, validEdit())
	if err != nil {
		t.Fatalf("UpdateSegment: %v", err)
	}
	got := saved.Segmen[1]
	want := Segment{Mulai: "00:05:10", Selesai: "00:05:50", Kategori: "mid", Label: "Team fight", Alasan: "first blood", Narasi: "Narasi baru"}
	if got != want {
		t.Errorf("segmen[1] = %+v, want %+v (alasan kept)", got, want)
	}
	if saved.Video != "source.mp4" || saved.Durasi != 3600 {
		t.Errorf("video/durasi not kept: %+v", saved)
	}

	jsonText := readProjectFile(t, st, storage.HighlightFile)
	if !strings.Contains(jsonText, "Team fight") || strings.Contains(jsonText, "First blood") {
		t.Errorf("highlight.json not rewritten: %s", jsonText)
	}
	narasi := readProjectFile(t, st, storage.NarrationFile)
	if !strings.Contains(narasi, "[00:05:10 - 00:05:50] Team fight\nNarasi baru") {
		t.Errorf("narasi.txt not rewritten: %q", narasi)
	}
}

func TestServiceUpdateSegment_ValidationErrorKeepsFiles(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*SegmentInput)
		field string
	}{
		{"bad time format", func(in *SegmentInput) { in.Mulai = "5:10" }, "mulai"},
		{"start after end", func(in *SegmentInput) { in.Mulai = "00:06:00" }, "mulai"},
		{"past video duration", func(in *SegmentInput) { in.Selesai = "02:00:00" }, "selesai"},
		{"overlaps previous segment", func(in *SegmentInput) { in.Mulai = "00:01:30" }, "mulai"},
		{"unknown category", func(in *SegmentInput) { in.Kategori = "gol" }, "kategori"},
		{"empty label", func(in *SegmentInput) { in.Label = "  " }, "label"},
		{"label too long", func(in *SegmentInput) { in.Label = "satu dua tiga empat lima enam tujuh delapan sembilan" }, "label"},
		{"empty narasi", func(in *SegmentInput) { in.Narasi = "" }, "narasi"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, st := newEditableService(t)
			beforeJSON := readProjectFile(t, st, storage.HighlightFile)
			beforeNarasi := readProjectFile(t, st, storage.NarrationFile)

			in := validEdit()
			tc.edit(&in)
			_, err := svc.UpdateSegment(context.Background(), testProjectID, 2, in)

			var appErr *httpx.AppError
			if !errors.As(err, &appErr) || appErr.Code != "highlight_invalid" {
				t.Fatalf("err = %v, want highlight_invalid", err)
			}
			details, _ := appErr.Details.([]ValidationError)
			found := false
			for _, d := range details {
				if d.Segmen == 2 && d.Field == tc.field {
					found = true
				}
			}
			if !found {
				t.Errorf("details = %+v, want an error on segmen 2 field %q", details, tc.field)
			}
			if readProjectFile(t, st, storage.HighlightFile) != beforeJSON {
				t.Error("highlight.json changed despite a validation error")
			}
			if readProjectFile(t, st, storage.NarrationFile) != beforeNarasi {
				t.Error("narasi.txt changed despite a validation error")
			}
		})
	}
}

func TestServiceUpdateSegment_UnknownSegment(t *testing.T) {
	svc, _ := newEditableService(t)
	for _, nomor := range []int{0, 3, -1} {
		_, err := svc.UpdateSegment(context.Background(), testProjectID, nomor, validEdit())
		var appErr *httpx.AppError
		if !errors.As(err, &appErr) || appErr.Code != "segment_not_found" {
			t.Errorf("nomor %d: err = %v, want segment_not_found", nomor, err)
		}
	}
}

func TestServiceUpdateSegment_NoHighlight(t *testing.T) {
	svc, _, _ := newTestService(t)
	_, err := svc.UpdateSegment(context.Background(), testProjectID, 1, validEdit())
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Code != "highlight_missing" {
		t.Errorf("err = %v, want highlight_missing", err)
	}
}

func TestServiceDeleteSegment_RewritesFiles(t *testing.T) {
	svc, st := newEditableService(t)

	saved, err := svc.DeleteSegment(context.Background(), testProjectID, 1)
	if err != nil {
		t.Fatalf("DeleteSegment: %v", err)
	}
	if len(saved.Segmen) != 1 || saved.Segmen[0].Label != "First blood" {
		t.Errorf("segmen = %+v, want only First blood", saved.Segmen)
	}
	if strings.Contains(readProjectFile(t, st, storage.HighlightFile), "Draft pick") {
		t.Error("highlight.json still has the deleted segment")
	}
	if strings.Contains(readProjectFile(t, st, storage.NarrationFile), "Narasi draft") {
		t.Error("narasi.txt still has the deleted segment")
	}
}

func TestServiceDeleteSegment_LastSegmentRejected(t *testing.T) {
	svc, st, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Save(ctx, testProjectID, validHighlightJSON); err != nil {
		t.Fatalf("Save: %v", err)
	}
	before := readProjectFile(t, st, storage.HighlightFile)

	_, err := svc.DeleteSegment(ctx, testProjectID, 1)
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Code != "highlight_invalid" {
		t.Fatalf("err = %v, want highlight_invalid", err)
	}
	if readProjectFile(t, st, storage.HighlightFile) != before {
		t.Error("highlight.json changed despite the rejected delete")
	}
}
