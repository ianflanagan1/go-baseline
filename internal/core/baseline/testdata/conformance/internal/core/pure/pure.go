// Package pure is a conformance fixture.
package pure

import (
	"strings"
)

// F is pure.
func F(s string) string { return strings.ToUpper(s) }
