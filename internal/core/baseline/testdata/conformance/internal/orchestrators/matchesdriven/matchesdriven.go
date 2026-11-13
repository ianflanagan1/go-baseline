// Package matchesdriven is a conformance fixture: an orchestrator uses
// errors.Is to compare an error with a driven adapter's sentinel error.
package matchesdriven

import (
	"errors"

	"example.com/conformance/internal/adapters/driven/notfound"
)

// Missing reports whether err says there is no such row.
func Missing(err error) bool { return errors.Is(err, notfound.ErrNotFound) }
