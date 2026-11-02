// Package clock is a conformance fixture.
package clock

import (
	"time"
)

// System is the system clock.
type System struct{}

// Now returns the time.
func (System) Now() time.Time { return time.Now() }
