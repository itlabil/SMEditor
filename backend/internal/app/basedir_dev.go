//go:build !embed_prod

package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// resolveBaseDir finds the repository root in dev builds by walking up from
// the working directory until a folder containing both backend/ and
// frontend/ is found. This works whether `go run` is started from the repo
// root or from backend/ (as the Makefile `dev` target does), without
// depending on where exactly the process happened to be started.
func resolveBaseDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	for {
		if isRepoRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repo root tidak ditemukan dari %s", dir)
		}
		dir = parent
	}
}

func isRepoRoot(dir string) bool {
	backend, err := os.Stat(filepath.Join(dir, "backend"))
	if err != nil || !backend.IsDir() {
		return false
	}
	frontend, err := os.Stat(filepath.Join(dir, "frontend"))
	return err == nil && frontend.IsDir()
}
