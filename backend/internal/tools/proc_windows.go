//go:build windows

package tools

import (
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// ConfigureProcessGroup sets cmd up so that canceling its context kills
// the whole process tree, not just the direct child. This matters because
// yt-dlp spawns ffmpeg as a child process; killing only the parent would
// leave ffmpeg running. See .agents/skills/sm-external-tools.
func ConfigureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	cmd.Cancel = func() error {
		return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
	}
	cmd.WaitDelay = 5 * time.Second
}
