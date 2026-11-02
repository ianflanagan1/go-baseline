// Package randomness is a conformance fixture.
package randomness

import (
	"math/rand/v2"
)

// F is random.
func F() int { return rand.IntN(6) }
