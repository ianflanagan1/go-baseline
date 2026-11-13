// Package emptybranch is a conformance fixture: a function checks whether
// the error a call returned is non-nil and does nothing in that branch.
package emptybranch

import "strconv"

// Parse returns text as a number, or 0.
func Parse(text string) int {
	n, err := strconv.Atoi(text)
	if err != nil {
	}
	return n
}
