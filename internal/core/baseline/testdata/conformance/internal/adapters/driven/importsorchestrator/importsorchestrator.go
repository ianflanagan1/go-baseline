// Package importsorchestrator is a conformance fixture.
package importsorchestrator

import (
	"example.com/conformance/internal/app"
)

// F calls an orchestrator.
func F() string { return app.Hello() }
