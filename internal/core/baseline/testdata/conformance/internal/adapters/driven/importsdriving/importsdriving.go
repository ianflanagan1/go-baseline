// Package importsdriving is a conformance fixture.
package importsdriving

import (
	"example.com/conformance/internal/adapters/driving/server"
)

// F uses a driving adapter's code.
func F() string { return server.Handle() }
