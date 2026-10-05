//go:build !windows

package tools

import (
	"os/exec"
	"syscall"
	"time"
)

// ConfigureProcessGroup sets cmd up so that canceling its context kills
// the whole process tree, not just the direct child. This matters because
// yt-dlp spawns ffmpeg as a child process; killing only the parent would
// leave ffmpeg running. See .agents/skills/sm-external-tools.
func ConfigureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
	cmd.WaitDelay = 5 * time.Second
}
