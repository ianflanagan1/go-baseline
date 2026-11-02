// Package importsdriving is a conformance fixture.
package importsdriving

import (
	"example.com/conformance/internal/adapters/driving/server"
)

// F calls a driving adapter.
func F() string { return server.Handle() }
