// Package errorreferencesorchestrator is a conformance fixture: an error
// type in value code references an orchestrator.
package errorreferencesorchestrator

import "example.com/conformance/internal/orchestrators/app"

// RefusedError says a greeting was refused.
type RefusedError struct{ greeting string }

// NewRefusedError returns the refusal of the orchestrator's greeting.
func NewRefusedError() RefusedError { return RefusedError{greeting: app.Hello()} }

// Error says what was refused.
func (e RefusedError) Error() string { return e.greeting + " was refused" }
