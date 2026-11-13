// Package stopwatch stands in for go-shared-code's system stopwatch, so the
// conformance fixture needs no network.
package stopwatch

import "time"

// System is the system stopwatch.
type System struct{}

// Start begins a measurement and returns a function that reports how long has
// passed since, each time it is called.
func (System) Start() func() time.Duration {
	start := time.Now()
	return func() time.Duration { return time.Since(start) }
}
