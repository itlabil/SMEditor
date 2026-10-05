package highlight

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"smeditor/internal/httpx"
	"smeditor/internal/modules/project"
	"smeditor/internal/storage"
)

const testProjectID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

type fakeProjectReader struct {
	p   *project.Project
	err error
}

func (f *fakeProjectReader) Get(ctx context.Context, id string) (*project.Project, error) {
	return f.p, f.err
}

type fakeProjectWriter struct {
	saved, cleared bool
}

func (f *fakeProjectWriter) SetHighlightSaved(ctx context.Context, id string) error {
	f.saved = true
	return nil
}

func (f *fakeProjectWriter) ClearHighlight(ctx context.Context, id string) error {
	f.cleared = true
	return nil
}

type fakeCategoryLookup struct {
	categories []string
}

func (f *fakeCategoryLookup) CategoriesForGame(ctx context.Context, gameCode string) ([]string, error) {
	return f.categories, nil
}

func newTestService(t *testing.T) (*Service, *storage.Storage, *fakeProjectWriter) {
	t.Helper()
	st := storage.New(t.TempDir())
	if _, err := st.EnsureProjectDir(testProjectID); err != nil {
		t.Fatalf("EnsureProjectDir: %v", err)
	}
	reader := &fakeProjectReader{p: &project.Project{
		ID: testProjectID, GameCode: "mlbb", VideoFile: "source.mp4", DurationSec: 3600,
	}}
	writer := &fakeProjectWriter{}
	categories := &fakeCategoryLookup{categories: mobaCategories}
	return NewService(st, reader, writer, categories), st, writer
}

const validHighlightJSON = `{
  "game": "mlbb",
  "ringkasan": "Tim A menang",
  "segmen": [
    {
      "mulai": "00:01:00",
      "selesai": "00:02:00",
      "kategori": "draft",
      "label": "Draft pick",
      "alasan": "caster membahas pick",
      "narasi": "Tim A mengamankan hero incaran"
    }
  ]
}`

func TestServiceSave_Success(t *testing.T) {
	svc, st, writer := newTestService(t)

	saved, err := svc.Save(context.Background(), testProjectID, validHighlightJSON)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if saved.Video != "source.mp4" || saved.Durasi != 3600 {
		t.Errorf("saved = %+v, want video=source.mp4 durasi=3600", saved)
	}
	if !writer.saved {
		t.Error("SetHighlightSaved was not called")
	}

	jsonPath, _ := st.FilePath(testProjectID, storage.HighlightFile)
	if _, err := os.Stat(jsonPath); err != nil {
		t.Errorf("highlight.json was not written: %v", err)
	}
	narasiPath, _ := st.FilePath(testProjectID, storage.NarrationFile)
	content, err := os.ReadFile(narasiPath)
	if err != nil {
		t.Fatalf("narasi.txt was not written: %v", err)
	}
	if !strings.Contains(string(content), "Tim A mengamankan hero incaran") {
		t.Errorf("narasi.txt = %q, want it to contain the narasi text", content)
	}
}

func TestServiceSave_AcceptsCodeFencedJSON(t *testing.T) {
	svc, _, _ := newTestService(t)
	fenced := "```json\n" + validHighlightJSON + "\n```"

	if _, err := svc.Save(context.Background(), testProjectID, fenced); err != nil {
		t.Fatalf("Save with fenced JSON: %v", err)
	}
}

func TestServiceSave_InvalidJSONWritesNothing(t *testing.T) {
	svc, st, writer := newTestService(t)

	_, err := svc.Save(context.Background(), testProjectID, "not json at all")
	if err == nil {
		t.Fatal("Save with invalid JSON: want error, got nil")
	}
	assertNoHighlightFilesWritten(t, st)
	if writer.saved {
		t.Error("SetHighlightSaved should not be called when parsing fails")
	}
}

func TestServiceSave_ValidationErrorsWriteNothing(t *testing.T) {
	svc, st, writer := newTestService(t)
	badJSON := `{"game":"mlbb","ringkasan":"x","segmen":[{"mulai":"bad","selesai":"00:02:00","kategori":"draft","label":"L","alasan":"a","narasi":"n"}]}`

	_, err := svc.Save(context.Background(), testProjectID, badJSON)
	if err == nil {
		t.Fatal("Save with invalid segment: want error, got nil")
	}
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Code != "highlight_invalid" {
		t.Errorf("err = %v, want highlight_invalid AppError", err)
	}
	details, ok := appErr.Details.([]ValidationError)
	if !ok || len(details) == 0 {
		t.Errorf("Details = %v, want a non-empty []ValidationError", appErr.Details)
	}
	assertNoHighlightFilesWritten(t, st)
	if writer.saved {
		t.Error("SetHighlightSaved should not be called when validation fails")
	}
}

func assertNoHighlightFilesWritten(t *testing.T, st *storage.Storage) {
	t.Helper()
	jsonPath, _ := st.FilePath(testProjectID, storage.HighlightFile)
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Error("highlight.json should not exist")
	}
	narasiPath, _ := st.FilePath(testProjectID, storage.NarrationFile)
	if _, err := os.Stat(narasiPath); !os.IsNotExist(err) {
		t.Error("narasi.txt should not exist")
	}
}

func TestServiceGet_MissingReturnsConflict(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.Get(context.Background(), testProjectID)
	if err == nil {
		t.Fatal("Get before saving: want error, got nil")
	}
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Code != "highlight_missing" {
		t.Errorf("err = %v, want highlight_missing", err)
	}
}

func TestServiceGet_ReturnsSavedHighlight(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Save(ctx, testProjectID, validHighlightJSON); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := svc.Get(ctx, testProjectID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Game != "mlbb" || len(got.Segmen) != 1 {
		t.Errorf("Get() = %+v", got)
	}
}

func TestServiceDelete_RemovesFilesAndClearsStatus(t *testing.T) {
	svc, st, writer := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Save(ctx, testProjectID, validHighlightJSON); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := svc.Delete(ctx, testProjectID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	assertNoHighlightFilesWritten(t, st)
	if !writer.cleared {
		t.Error("ClearHighlight was not called")
	}
}

func TestServiceDelete_NoSavedHighlightIsNotAnError(t *testing.T) {
	svc, _, writer := newTestService(t)

	if err := svc.Delete(context.Background(), testProjectID); err != nil {
		t.Fatalf("Delete with nothing saved: %v", err)
	}
	if !writer.cleared {
		t.Error("ClearHighlight was not called")
	}
}

func TestServiceNarasiPath_MissingReturnsConflict(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.NarasiPath(context.Background(), testProjectID)
	if err == nil {
		t.Fatal("NarasiPath before saving: want error, got nil")
	}
}

func TestServiceNarasiPath_ReturnsPathAfterSave(t *testing.T) {
	svc, st, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.Save(ctx, testProjectID, validHighlightJSON); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := svc.NarasiPath(ctx, testProjectID)
	if err != nil {
		t.Fatalf("NarasiPath: %v", err)
	}
	want, _ := st.FilePath(testProjectID, storage.NarrationFile)
	if got != want {
		t.Errorf("NarasiPath = %q, want %q", got, want)
	}
}
