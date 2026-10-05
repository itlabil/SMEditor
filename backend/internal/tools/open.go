package tools

import "context"

// Opener opens a folder in the OS's desktop file manager (Explorer on
// Windows, xdg-open on Linux), per docs/sequence.md
// ("POST /api/projects/:id/open-folder").
type Opener struct{}

func (Opener) OpenFolder(ctx context.Context, path string) error {
	return openFolder(ctx, path)
}
