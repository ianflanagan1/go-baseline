// Package unformatted is a conformance fixture: it is not as gofumpt and
// goimports write it.
package unformatted

import (
	"strings"
	"fmt"
)

// Shout shouts s.
func Shout(s string) string {
    return fmt.Sprint(strings.ToUpper(s))
}
