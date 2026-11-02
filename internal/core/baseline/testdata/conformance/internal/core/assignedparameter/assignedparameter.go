// Package assignedparameter is a conformance fixture.
package assignedparameter

// F assigns a new value to its parameter.
func F(n int) int {
	n = 2
	return n
}
