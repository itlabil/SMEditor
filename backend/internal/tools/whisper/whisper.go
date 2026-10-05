// Package whisper calls the local Whisper binary.
package whisper

import (
	"context"
	"strings"

	"smeditor/internal/tools"
)

type Client struct{}

func (Client) Check(ctx context.Context, path string) (resolvedPath string, found bool, version string) {
	res := tools.CheckExecutable(ctx, path, []string{"--version"}, parseVersion)
	return res.Path, res.Found, res.Version
}

// parseVersion keeps only the first line of the tool's output; whisper.cpp
// builds vary in what --version prints, so this is best-effort.
func parseVersion(output string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(output), "\n")
	return strings.TrimSpace(line)
}
