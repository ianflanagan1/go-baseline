// Package configuration is a conformance fixture.
package configuration

import (
	"os"
)

// F reads the environment.
func F() string { return os.Getenv("HOME") }
