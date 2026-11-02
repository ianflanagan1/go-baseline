// Package filesystem is a conformance fixture.
package filesystem

import (
	"os"
)

// F reads a file.
func F() ([]byte, error) { return os.ReadFile("f") }
