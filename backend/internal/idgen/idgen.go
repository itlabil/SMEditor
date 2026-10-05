// Package idgen generates the ULIDs used as primary keys across the app,
// per .agents/rules/conventions.md ("ID memakai ULID").
package idgen

import (
	"crypto/rand"

	"github.com/oklog/ulid/v2"
)

// New returns a new, lexicographically sortable ULID string.
func New() string {
	return ulid.MustNew(ulid.Now(), rand.Reader).String()
}
