// Package importsdriven is a conformance fixture.
package importsdriven

import (
	"example.com/conformance/internal/adapters/driven/clock"
)

// Handle answers a request by reading a driven adapter itself.
func Handle() int { return clock.System{}.Now().Year() }
