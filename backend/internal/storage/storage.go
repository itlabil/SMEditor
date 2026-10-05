// Package storage is the only place that composes paths inside a project's
// data folder, per .agents/rules/architecture.md.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// File names inside data/projects/{id}/, per docs/prd.md and docs/flow.md.
const (
	SourceVideoFile    = "source.mp4"
	ProxyVideoFile     = "proxy.mp4"
	AudioFile          = "audio.wav"
	TranscriptJSONFile = "transcript.json"
	TranscriptTXTFile  = "transcript.txt"
	HighlightFile      = "highlight.json"
	NarrationFile      = "narasi.txt"
	ThumbnailFile      = "thumbnail.jpg"
)

var ulidPattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// ValidID reports whether id is a well-formed ULID. Any path built from a
// project id must check this first, per .agents/rules/conventions.md ("ID
// project dari URL divalidasi sebagai ULID sebelum dipakai menyusun path").
func ValidID(id string) bool {
	return ulidPattern.MatchString(id)
}

// Storage composes paths under a single projects directory.
type Storage struct {
	projectsDir string
}

// New creates a Storage rooted at dataDir/projects.
func New(dataDir string) *Storage {
	return &Storage{projectsDir: filepath.Join(dataDir, "projects")}
}

// ProjectDir returns the absolute folder for a project.
func (s *Storage) ProjectDir(id string) (string, error) {
	if !ValidID(id) {
		return "", fmt.Errorf("invalid project id %q", id)
	}
	return filepath.Join(s.projectsDir, id), nil
}

// FilePath returns the absolute path of a named file inside a project's
// folder. filename should be one of the File constants above.
func (s *Storage) FilePath(id, filename string) (string, error) {
	dir, err := s.ProjectDir(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filename), nil
}

// EnsureProjectDir creates a project's folder if it does not exist yet and
// returns its absolute path.
func (s *Storage) EnsureProjectDir(id string) (string, error) {
	dir, err := s.ProjectDir(id)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create project dir %s: %w", dir, err)
	}
	return dir, nil
}

// DeleteProjectDir removes a project's entire folder. id is validated as a
// ULID first, so this can only ever target a path inside projectsDir, per
// .agents/rules/conventions.md ("Hapus folder hanya boleh menyasar path di
// dalam data/projects/").
func (s *Storage) DeleteProjectDir(id string) error {
	dir, err := s.ProjectDir(id)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove project dir %s: %w", dir, err)
	}
	return nil
}

// Stat reports whether a named file inside a project's folder exists.
func (s *Storage) Stat(id, filename string) bool {
	path, err := s.FilePath(id, filename)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
