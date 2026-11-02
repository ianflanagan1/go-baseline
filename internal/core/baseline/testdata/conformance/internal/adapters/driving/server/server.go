// Package server is a conformance fixture: a driving adapter that keeps to
// its role, calling an orchestrator.
package server

import (
	"example.com/conformance/internal/app"
)

// Handle answers a request by calling an orchestrator.
func Handle() string { return app.Hello() }
