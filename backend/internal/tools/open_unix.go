//go:build !windows

package tools

import (
	"context"
	"os/exec"
)

// openFolder launches the desktop file manager at path. It only returns
// an error if the command could not even be started; a GUI app's own
// exit code/lifetime is irrelevant to the caller.
func openFolder(ctx context.Context, path string) error {
	return exec.CommandContext(ctx, "xdg-open", path).Start()
}
