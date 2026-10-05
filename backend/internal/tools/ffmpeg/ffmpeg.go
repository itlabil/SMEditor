// Package ffmpeg calls the ffmpeg and ffprobe binaries.
package ffmpeg

import (
	"context"

	"smeditor/internal/tools"
)

type Ffmpeg struct{}

func (Ffmpeg) Check(ctx context.Context, path string) (resolvedPath string, found bool, version string) {
	res := tools.CheckExecutable(ctx, path, []string{"-version"}, parseVersion)
	return res.Path, res.Found, res.Version
}

type Ffprobe struct{}

func (Ffprobe) Check(ctx context.Context, path string) (resolvedPath string, found bool, version string) {
	res := tools.CheckExecutable(ctx, path, []string{"-version"}, parseVersion)
	return res.Path, res.Found, res.Version
}
