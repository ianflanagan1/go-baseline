// Package droppederror is a conformance fixture: a function that returns an
// error, called as a statement.
package droppederror

import "errors"

func validate(s string) error {
	if s == "" {
		return errors.New("empty")
	}
	return nil
}

// Use drops validate's error.
func Use(s string) string {
	validate(s)
	return s
}
