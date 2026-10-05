// Package tools is the only place internal packages call exec.Command, per
// .agents/rules/architecture.md. Other modules reach it through a small
// interface defined in their own package.
package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// BaseDir is the fixed root that relative tool paths resolve against. It is
// set once in app.Run() from the resolved project root (the same base used
// for DataDir), never the process's current working directory, so a path
// like "tools/yt-dlp" means the same file regardless of where the binary
// was started from.
var BaseDir string

// ResolveExecutable finds the real file for a configured tool path:
//
//   - a bare name with no folder separator (e.g. "ffmpeg") is searched in
//     the system PATH with exec.LookPath, which also handles PATHEXT/.exe
//     on Windows.
//   - an absolute path is used as-is.
//   - a relative path (e.g. "tools/yt-dlp") is joined to BaseDir.
//
// found reports whether the resolved file actually exists (for a PATH
// lookup, whether exec.LookPath found it).
func ResolveExecutable(path string) (resolved string, found bool) {
	if path == "" {
		return "", false
	}

	if filepath.Base(path) == path {
		p, err := exec.LookPath(path)
		if err != nil {
			return path, false
		}
		return p, true
	}

	resolved = path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(BaseDir, resolved)
	}
	resolved = withPlatformExtension(resolved)

	if _, err := os.Stat(resolved); err != nil {
		return resolved, false
	}
	return resolved, true
}

// withPlatformExtension appends ".exe" on Windows when the path has no
// extension yet, so settings never need an OS-specific value.
func withPlatformExtension(path string) string {
	if runtime.GOOS != "windows" || filepath.Ext(path) != "" {
		return path
	}
	return path + ".exe"
}
