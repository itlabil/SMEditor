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

// CheckModel reports whether the configured model file exists. Unlike
// Check, it never runs anything: a model file has no --version to call.
func (Client) CheckModel(ctx context.Context, path string) (resolvedPath string, found bool, version string) {
	resolvedPath, found = tools.ResolveExecutable(path)
	return resolvedPath, found, ""
}

// parseVersion keeps only the first line of the tool's output; whisper.cpp
// builds vary in what --version prints, so this is best-effort.
func parseVersion(output string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(output), "\n")
	return strings.TrimSpace(line)
}
