//go:build !embed_prod

package webdist

import "io/fs"

// Dist is nil in dev builds; the frontend is served by the Vite dev server,
// which proxies /api to this backend.
var Dist fs.FS = nil

const Embedded = false
