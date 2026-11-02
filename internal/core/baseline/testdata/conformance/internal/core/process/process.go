// Package process is a conformance fixture.
package process

import (
	"os/exec"
)

// F runs a process.
func F() error { return exec.Command("true").Run() }
