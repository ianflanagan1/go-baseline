// Package matchesdriven is a conformance fixture: a driving adapter matches
// a driven adapter's not-found error, to answer 404.
package matchesdriven

import (
	"errors"

	"example.com/conformance/internal/adapters/driven/notfound"
)

// Status returns the HTTP status for err.
func Status(err error) int {
	if errors.Is(err, notfound.ErrNotFound) {
		return 404
	}
	return 500
}
