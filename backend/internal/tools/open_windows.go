//go:build windows

package tools

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// openFolderTimeout bounds how long the explorer.exe process we start is
// allowed to keep running before it gets reaped. explorer.exe hands the
// path to the already-running shell and returns almost immediately, so
// this is generous headroom, not a normal completion time.
const openFolderTimeout = 10 * time.Second

// openFolder launches Explorer at path.
//
// The caller's ctx (the HTTP request context) is deliberately not used to
// bound this command: net/http cancels c.Request.Context() the instant
// the handler returns, which happens right after the command is started,
// so wiring it straight into exec.CommandContext would race explorer.exe
// against its own kill-on-cancel watcher and risk killing the child
// before it finished opening. ctx is stripped of cancellation here and
// replaced with an independent timeout instead.
//
// It only returns an error if the command could not even be started:
// explorer.exe's own exit code is notoriously unreliable (often nonzero
// on success), so it is never checked, and the caller does not wait for
// the window's lifetime -- Wait() just runs in the background to reap
// the process once it (or the timeout) ends.
func openFolder(ctx context.Context, path string) error {
	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openFolderTimeout)
	cmd := exec.CommandContext(runCtx, "explorer", path)
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("explorer: %w", err)
	}
	go func() {
		defer cancel()
		_ = cmd.Wait()
	}()
	return nil
}
