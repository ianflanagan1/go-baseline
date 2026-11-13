// Package notfound is a conformance fixture: a driven adapter with a
// sentinel error of its own.
package notfound

import "errors"

// ErrNotFound is returned when there is no such row.
var ErrNotFound = errors.New("no such row")
