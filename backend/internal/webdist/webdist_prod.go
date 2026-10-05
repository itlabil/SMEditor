//go:build embed_prod

package webdist

import (
	"embed"
	"io/fs"
)

//go:embed dist
var distFS embed.FS

// Dist holds the built Vue app, populated by `make build` before compiling
// with the embed_prod tag.
var Dist, _ = fs.Sub(distFS, "dist")

const Embedded = true
