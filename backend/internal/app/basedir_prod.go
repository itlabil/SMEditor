//go:build embed_prod

package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// resolveBaseDir returns the folder containing the running binary. Release
// builds ship with tools/ and data/ as siblings of smeditor.exe.
func resolveBaseDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("os.Executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe), nil
}
