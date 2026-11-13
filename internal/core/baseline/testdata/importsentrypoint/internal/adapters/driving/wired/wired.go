// Package wired references the entrypoint.
package wired

import "example.com/importsentrypoint/cmd/app"

// Name names the program.
func Name() string { return app.Name() }
