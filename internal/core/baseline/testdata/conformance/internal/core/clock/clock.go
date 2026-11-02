// Package clock is a conformance fixture.
package clock

import (
	"time"
)

// F reads the clock.
func F() int { return time.Now().Year() }
