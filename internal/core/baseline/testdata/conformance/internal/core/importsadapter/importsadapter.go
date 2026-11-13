// Package importsadapter is a conformance fixture.
package importsadapter

import (
	"example.com/conformance/internal/adapters/driven/notes"
)

// F calls an adapter.
func F() (string, error) { return notes.Store{}.Read("note.txt") }
