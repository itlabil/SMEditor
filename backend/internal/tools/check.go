package tools

import (
	"context"
	"os/exec"
)

// CheckResult reports whether a configured tool was found and, best-effort,
// its version string.
type CheckResult struct {
	Path    string
	Found   bool
	Version string
	// Output is the raw combined stdout/stderr of the version call, for
	// tools that report more than a version there (whisper prints which
	// compute backend it loaded).
	Output string
}

// CheckExecutable resolves path and, if found, runs it with versionArgs to
// extract a version string via parseVersion. A failure to run the tool
// (unexpected exit code, no such flag, ...) does not clear Found; Version
// is simply left empty in that case.
func CheckExecutable(ctx context.Context, path string, versionArgs []string, parseVersion func(string) string) CheckResult {
	resolved, found := ResolveExecutable(path)
	if !found {
		return CheckResult{Path: resolved}
	}

	out, _ := exec.CommandContext(ctx, resolved, versionArgs...).CombinedOutput()
	return CheckResult{Path: resolved, Found: true, Version: parseVersion(string(out)), Output: string(out)}
}
