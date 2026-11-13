// Package importsdriven is a conformance fixture.
package importsdriven

import (
	"example.com/conformance/internal/adapters/driven/notes"
)

// Handle answers a request by reading a driven adapter itself.
func Handle() (string, error) { return notes.Store{}.Read("note.txt") }
