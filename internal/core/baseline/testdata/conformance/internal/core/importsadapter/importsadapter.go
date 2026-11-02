// Package importsadapter is a conformance fixture.
package importsadapter

import (
	"example.com/conformance/internal/adapters/driven/clock"
)

// F calls an adapter.
func F() int { return clock.System{}.Now().Year() }
