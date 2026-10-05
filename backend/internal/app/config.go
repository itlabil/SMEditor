package app

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port string
	// BaseDir is the resolved, absolute project root. Relative tool paths
	// in settings and the default DataDir resolve against it, never
	// against the process's working directory.
	BaseDir string
	DataDir string
}

func LoadConfig() (Config, error) {
	port := os.Getenv("SMEDITOR_PORT")
	if port == "" {
		port = "8080"
	}

	baseDir := os.Getenv("SMEDITOR_BASE_DIR")
	if baseDir == "" {
		resolved, err := resolveBaseDir()
		if err != nil {
			return Config{}, fmt.Errorf("resolve base dir: %w", err)
		}
		baseDir = resolved
	}

	dataDir := os.Getenv("SMEDITOR_DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	return Config{Port: port, BaseDir: baseDir, DataDir: dataDir}, nil
}

// DataPath returns the absolute data directory: DataDir as-is if absolute,
// otherwise DataDir joined to BaseDir.
func (c Config) DataPath() string {
	if filepath.IsAbs(c.DataDir) {
		return c.DataDir
	}
	return filepath.Join(c.BaseDir, c.DataDir)
}
