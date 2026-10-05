// Package whisper calls the local Whisper binary.
package whisper

import (
	"context"
	"regexp"
	"strings"

	"smeditor/internal/tools"
)

// Compute backends reported by CheckWithBackend.
const (
	BackendGPU = "gpu"
	BackendCPU = "cpu"
)

type Client struct{}

func (c Client) Check(ctx context.Context, path string) (resolvedPath string, found bool, version string) {
	resolvedPath, found, version, _, _ = c.CheckWithBackend(ctx, path)
	return resolvedPath, found, version
}

// CheckWithBackend runs whisper-cli --version once and also reports which
// compute backend it loaded: BackendGPU, BackendCPU, or "" when the
// output does not say (older builds print no backend lines). gpu is the
// CUDA device name when one was found.
func (Client) CheckWithBackend(ctx context.Context, path string) (resolvedPath string, found bool, version, backend, gpu string) {
	res := tools.CheckExecutable(ctx, path, []string{"--version"}, parseVersion)
	if !res.Found {
		return res.Path, false, "", "", ""
	}
	backend, gpu = parseBackend(res.Output)
	return res.Path, true, res.Version, backend, gpu
}

// CheckModel reports whether the configured model file exists. Unlike
// Check, it never runs anything: a model file has no --version to call.
func (Client) CheckModel(ctx context.Context, path string) (resolvedPath string, found bool, version string) {
	resolvedPath, found = tools.ResolveExecutable(path)
	return resolvedPath, found, ""
}

// parseVersion returns the line that names the version. CUDA builds print
// device and backend-loading lines before it, so the first line alone is
// not the version; whisper.cpp builds vary, so this stays best-effort and
// falls back to the first line.
func parseVersion(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "version") {
			return strings.TrimSpace(line)
		}
	}
	return strings.TrimSpace(lines[0])
}

var (
	cudaDevicePattern = regexp.MustCompile(`Device \d+: ([^,\r\n]+)`)
	usingCUDAPattern  = regexp.MustCompile(`using CUDA\d* backend`)
)

// parseBackend reads whisper.cpp's startup lines:
//   - "no GPU found" (printed at model load) means CPU, even if the CUDA
//     library itself loaded;
//   - "using CUDA0 backend" (model load) or "loaded CUDA backend"
//     (--version and every run) means GPU;
//   - only "loaded CPU backend" means CPU: a CPU-only build, or
//     ggml-cuda.dll failed to load (e.g. a missing CUDA runtime DLL) and
//     whisper fell back to CPU without an error.
func parseBackend(output string) (backend, gpu string) {
	switch {
	case strings.Contains(output, "no GPU found"):
		return BackendCPU, ""
	case usingCUDAPattern.MatchString(output), strings.Contains(output, "loaded CUDA backend"):
		if m := cudaDevicePattern.FindStringSubmatch(output); m != nil {
			gpu = strings.TrimSpace(m[1])
		}
		return BackendGPU, gpu
	case strings.Contains(output, "loaded CPU backend"):
		return BackendCPU, ""
	}
	return "", ""
}
