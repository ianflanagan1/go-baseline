// Package wiring is a conformance fixture: an entrypoint helper that isn't
// under cmd/internal/, so Go lets other roles import it.
package wiring

// Name names the program.
func Name() string { return "app" }
