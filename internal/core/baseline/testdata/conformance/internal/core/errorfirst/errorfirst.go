// Package errorfirst is a conformance fixture: a function returns its error
// first.
package errorfirst

import "errors"

// ErrEmpty is returned for an empty name.
var ErrEmpty = errors.New("the name is empty")

// Length returns the length of name, after the error.
func Length(name string) (error, int) {
	if name == "" {
		return ErrEmpty, 0
	}
	return nil, len(name)
}
