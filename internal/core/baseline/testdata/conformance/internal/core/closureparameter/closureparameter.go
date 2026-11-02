// Package closureparameter is a conformance fixture.
package closureparameter

// F has a closure that assigns to F's parameter.
func F(n int) int {
	reset := func() { n = 0 }
	reset()
	return n
}
