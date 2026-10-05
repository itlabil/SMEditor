package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validID = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

func TestValidID(t *testing.T) {
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"valid ulid", validID, true},
		{"path traversal", "../..", false},
		{"path traversal with segment", "../../etc/passwd", false},
		{"empty", "", false},
		{"too short", "01ARZ3", false},
		{"lowercase", strings.ToLower(validID), false},
		{"contains slash", "01ARZ3NDEKTSV4RRFFQ69G5F/V", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidID(tc.id); got != tc.want {
				t.Errorf("ValidID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

func TestProjectDirRejectsInvalidID(t *testing.T) {
	s := New(t.TempDir())

	for _, id := range []string{"../..", "../../etc", "", "not-a-ulid"} {
		if _, err := s.ProjectDir(id); err == nil {
			t.Errorf("ProjectDir(%q): want error, got nil", id)
		}
	}
}

func TestProjectDirStaysInsideProjectsDir(t *testing.T) {
	base := t.TempDir()
	s := New(base)

	dir, err := s.ProjectDir(validID)
	if err != nil {
		t.Fatalf("ProjectDir: %v", err)
	}
	wantPrefix := filepath.Join(base, "projects") + string(filepath.Separator)
	if !strings.HasPrefix(dir, wantPrefix) {
		t.Errorf("ProjectDir = %q, want prefix %q", dir, wantPrefix)
	}
}

func TestEnsureAndDeleteProjectDir(t *testing.T) {
	s := New(t.TempDir())

	dir, err := s.EnsureProjectDir(validID)
	if err != nil {
		t.Fatalf("EnsureProjectDir: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("project dir %s was not created", dir)
	}

	if err := s.DeleteProjectDir(validID); err != nil {
		t.Fatalf("DeleteProjectDir: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("project dir %s still exists after delete", dir)
	}
}

func TestDeleteProjectDirRejectsInvalidID(t *testing.T) {
	s := New(t.TempDir())

	if err := s.DeleteProjectDir("../.."); err == nil {
		t.Fatal("DeleteProjectDir(\"../..\"): want error, got nil")
	}
}

func TestFilePath(t *testing.T) {
	base := t.TempDir()
	s := New(base)

	got, err := s.FilePath(validID, SourceVideoFile)
	if err != nil {
		t.Fatalf("FilePath: %v", err)
	}
	want := filepath.Join(base, "projects", validID, SourceVideoFile)
	if got != want {
		t.Errorf("FilePath = %q, want %q", got, want)
	}
}

func TestStat(t *testing.T) {
	s := New(t.TempDir())
	if _, err := s.EnsureProjectDir(validID); err != nil {
		t.Fatalf("EnsureProjectDir: %v", err)
	}
	path, err := s.FilePath(validID, ThumbnailFile)
	if err != nil {
		t.Fatalf("FilePath: %v", err)
	}

	if s.Stat(validID, ThumbnailFile) {
		t.Error("Stat() = true before file exists")
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !s.Stat(validID, ThumbnailFile) {
		t.Error("Stat() = false after file was created")
	}
}
