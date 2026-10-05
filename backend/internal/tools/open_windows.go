//go:build windows

package tools

import (
	"context"
	"os/exec"
)

// openFolder launches Explorer at path. It only returns an error if the
// command could not even be started: explorer.exe's own exit code is
// notoriously unreliable (often nonzero on success), so it is never
// checked, and the caller does not wait for the window's lifetime.
func openFolder(ctx context.Context, path string) error {
	return exec.CommandContext(ctx, "explorer", path).Start()
}
