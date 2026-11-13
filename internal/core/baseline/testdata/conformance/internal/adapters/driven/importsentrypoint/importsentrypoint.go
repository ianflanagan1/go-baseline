// Package importsentrypoint is a conformance fixture: a driven adapter
// references entrypoint code.
package importsentrypoint

import "example.com/conformance/cmd/wiring"

// Name names the program.
func Name() string { return wiring.Name() }
