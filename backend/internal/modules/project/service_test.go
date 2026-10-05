package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"smeditor/internal/db"
	"smeditor/internal/httpx"
	"smeditor/internal/storage"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(context.Background(), conn); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}

	repo := NewRepository(conn)
	st := storage.New(t.TempDir())
	return NewService(repo, st)
}

func validRequest() CreateRequest {
	return CreateRequest{
		Name:       "MPL Game 3",
		GameCode:   "mlbb",
		YoutubeURL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	}
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error %v is not an *httpx.AppError", err)
	}
	return appErr.Code
}

func TestServiceCreate_Success(t *testing.T) {
	svc := newTestService(t)

	p, err := svc.Create(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Status != StatusBaru {
		t.Errorf("Status = %q, want %q", p.Status, StatusBaru)
	}
	if !storage.ValidID(p.ID) {
		t.Errorf("ID %q is not a valid ULID", p.ID)
	}

	dir, err := svc.storage.ProjectDir(p.ID)
	if err != nil {
		t.Fatalf("ProjectDir: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("project dir %s was not created", dir)
	}
}

func TestServiceCreate_NameRequired(t *testing.T) {
	svc := newTestService(t)
	req := validRequest()
	req.Name = "   "

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("Create with empty name: want error, got nil")
	}
	if code := appErrCode(t, err); code != "name_required" {
		t.Errorf("code = %q, want name_required", code)
	}
}

func TestServiceCreate_InvalidURL(t *testing.T) {
	svc := newTestService(t)
	req := validRequest()
	req.YoutubeURL = "https://vimeo.com/12345"

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("Create with non-YouTube URL: want error, got nil")
	}
	if code := appErrCode(t, err); code != "invalid_url" {
		t.Errorf("code = %q, want invalid_url", code)
	}
}

func TestServiceCreate_GameNotFound(t *testing.T) {
	svc := newTestService(t)
	req := validRequest()
	req.GameCode = "not-a-real-game"

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("Create with unknown game code: want error, got nil")
	}
	if code := appErrCode(t, err); code != "game_not_found" {
		t.Errorf("code = %q, want game_not_found", code)
	}
}

func TestServiceGet_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Get(context.Background(), "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err == nil {
		t.Fatal("Get with unknown id: want error, got nil")
	}
	if code := appErrCode(t, err); code != "project_not_found" {
		t.Errorf("code = %q, want project_not_found", code)
	}
}

func TestServiceGet_InvalidID(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Get(context.Background(), "../..")
	if err == nil {
		t.Fatal("Get with path-traversal id: want error, got nil")
	}
	if code := appErrCode(t, err); code != "invalid_id" {
		t.Errorf("code = %q, want invalid_id", code)
	}
}

func TestServiceList_ReturnsCreatedProjects(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, validRequest()); err != nil {
		t.Fatalf("Create: %v", err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("List() len = %d, want 1", len(list))
	}
	if list[0].Name != "MPL Game 3" {
		t.Errorf("List()[0].Name = %q, want %q", list[0].Name, "MPL Game 3")
	}
}

func TestServiceDelete_RemovesRowAndFolder(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, validRequest())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	dir, err := svc.storage.ProjectDir(p.ID)
	if err != nil {
		t.Fatalf("ProjectDir: %v", err)
	}

	if err := svc.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("project dir %s still exists after delete", dir)
	}
	if _, err := svc.Get(ctx, p.ID); err == nil {
		t.Fatal("Get after Delete: want error, got nil")
	}
}

func TestServiceDelete_NotFound(t *testing.T) {
	svc := newTestService(t)

	err := svc.Delete(context.Background(), "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err == nil {
		t.Fatal("Delete with unknown id: want error, got nil")
	}
	if code := appErrCode(t, err); code != "project_not_found" {
		t.Errorf("code = %q, want project_not_found", code)
	}
}

func TestServiceDelete_RejectsPathTraversalID(t *testing.T) {
	svc := newTestService(t)

	err := svc.Delete(context.Background(), "../..")
	if err == nil {
		t.Fatal("Delete with path-traversal id: want error, got nil")
	}
	if code := appErrCode(t, err); code != "invalid_id" {
		t.Errorf("code = %q, want invalid_id", code)
	}
}
