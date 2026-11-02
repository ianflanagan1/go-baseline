// Package globalstate is a conformance fixture.
package globalstate

var count int

// F counts.
func F() int {
	count++
	return count
}
