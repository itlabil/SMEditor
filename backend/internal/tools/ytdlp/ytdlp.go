// Package ytdlp calls the yt-dlp binary.
package ytdlp

import (
	"context"

	"smeditor/internal/tools"
)

type Client struct{}

func (Client) Check(ctx context.Context, path string) (resolvedPath string, found bool, version string) {
	res := tools.CheckExecutable(ctx, path, []string{"--version"}, parseVersion)
	return res.Path, res.Found, res.Version
}
