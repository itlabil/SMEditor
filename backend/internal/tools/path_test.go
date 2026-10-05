package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// anyPathBinary returns the name of a binary that is reliably present in
// PATH on the OS running the test, so the bare-name case can be exercised
// without depending on a specific tool being installed.
func anyPathBinary(t *testing.T) string {
	t.Helper()
	candidates := []string{"sh", "ls", "cmd", "cmd.exe"}
	if runtime.GOOS == "windows" {
		candidates = []string{"cmd", "cmd.exe", "where"}
	}
	for _, name := range candidates {
		if p, err := exec.LookPath(name); err == nil {
			return filepath.Base(p)
		}
	}
	t.Skip("tidak ada binary kandidat di PATH untuk tes ini")
	return ""
}

func TestResolveExecutable_BareNameFoundInPath(t *testing.T) {
	name := anyPathBinary(t)
	want, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("exec.LookPath(%q): %v", name, err)
	}

	resolved, found := ResolveExecutable(name)
	if !found {
		t.Fatalf("ResolveExecutable(%q) found = false, want true", name)
	}
	if resolved != want {
		t.Errorf("ResolveExecutable(%q) = %q, want %q", name, resolved, want)
	}
}

func TestResolveExecutable_BareNameNotFoundInPath(t *testing.T) {
	const name = "smeditor-definitely-not-a-real-binary"

	resolved, found := ResolveExecutable(name)
	if found {
		t.Fatalf("ResolveExecutable(%q) found = true, want false", name)
	}
	if resolved != name {
		t.Errorf("ResolveExecutable(%q) resolved = %q, want %q unchanged", name, resolved, name)
	}
}

func TestResolveExecutable_RelativePathJoinsBaseDir(t *testing.T) {
	dir := t.TempDir()
	old := BaseDir
	BaseDir = dir
	t.Cleanup(func() { BaseDir = old })

	toolPath := filepath.Join("tools", "fake-tool")
	full := filepath.Join(dir, toolPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	resolved, found := ResolveExecutable(toolPath)
	if !found {
		t.Fatalf("ResolveExecutable(%q) found = false, want true", toolPath)
	}
	wantResolved := full
	if runtime.GOOS == "windows" && filepath.Ext(wantResolved) == "" {
		wantResolved += ".exe"
	}
	if resolved != wantResolved {
		t.Errorf("ResolveExecutable(%q) = %q, want %q", toolPath, resolved, wantResolved)
	}

	// Changing the working directory must not affect resolution: it is
	// anchored to BaseDir, not cwd.
	otherCwd := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(otherCwd); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })

	resolvedAgain, foundAgain := ResolveExecutable(toolPath)
	if !foundAgain || resolvedAgain != wantResolved {
		t.Errorf("ResolveExecutable(%q) after chdir = (%q, %v), want (%q, true)", toolPath, resolvedAgain, foundAgain, wantResolved)
	}
}

func TestResolveExecutable_RelativePathNotFound(t *testing.T) {
	dir := t.TempDir()
	old := BaseDir
	BaseDir = dir
	t.Cleanup(func() { BaseDir = old })

	resolved, found := ResolveExecutable(filepath.Join("tools", "missing-tool"))
	if found {
		t.Fatalf("ResolveExecutable found = true for a file that was never created")
	}
	if filepath.Dir(resolved) != filepath.Join(dir, "tools") {
		t.Errorf("resolved path %q is not under BaseDir %q", resolved, dir)
	}
}

func TestResolveExecutable_AbsolutePathUsedAsIs(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "fake-tool")
	if err := os.WriteFile(full, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	resolved, found := ResolveExecutable(full)
	if !found {
		t.Fatalf("ResolveExecutable(%q) found = false, want true", full)
	}
	if resolved != full {
		t.Errorf("ResolveExecutable(%q) = %q, want unchanged %q", full, resolved, full)
	}
}

func TestResolveExecutable_AbsolutePathNotFound(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "missing-tool")

	resolved, found := ResolveExecutable(full)
	if found {
		t.Fatalf("ResolveExecutable(%q) found = true for a file that does not exist", full)
	}
	if resolved != full {
		t.Errorf("ResolveExecutable(%q) = %q, want unchanged %q", full, resolved, full)
	}
}
