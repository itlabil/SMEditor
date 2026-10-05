//go:build !windows

package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// openFolderTimeout bounds how long xdg-open itself may run. xdg-open
// typically hands the path to the desktop's already-running file manager
// (e.g. "gio open") and exits almost immediately, so this is generous
// headroom, not a normal completion time.
const openFolderTimeout = 10 * time.Second

// openFolder launches the desktop file manager at path via xdg-open.
//
// The caller's ctx (the HTTP request context) is deliberately not used to
// bound this command: net/http cancels c.Request.Context() the instant
// the handler returns, which happens right after the command would be
// started, so wiring it straight into exec.CommandContext raced xdg-open
// against its own kill-on-cancel watcher and silently killed the child
// before it finished handing off to the file manager -- the same command
// run from a terminal (no such context) always worked. ctx is stripped of
// cancellation here and replaced with an independent timeout instead.
//
// Unlike a plain Start(), this waits for xdg-open's own exit and reports
// its exit code/stderr as a real error, so a failure (no handler
// registered, broken $DISPLAY, etc.) is never swallowed.
func openFolder(ctx context.Context, path string) error {
	runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openFolderTimeout)
	defer cancel()

	out, err := exec.CommandContext(runCtx, "xdg-open", path).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("xdg-open: %w (%s)", err, msg)
		}
		return fmt.Errorf("xdg-open: %w", err)
	}
	return nil
}
