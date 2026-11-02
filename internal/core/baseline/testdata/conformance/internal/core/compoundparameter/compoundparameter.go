// Package compoundparameter is a conformance fixture.
package compoundparameter

// F updates its parameter with a compound assignment.
func F(s string) string {
	s += "!"
	return s
}
