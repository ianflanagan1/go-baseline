// Package rows is a conformance fixture: a helper that maps rows for two
// database stores references an orchestrator.
package rows

import "example.com/conformance/internal/orchestrators/app"

// Greeting maps a row.
func Greeting(row string) string { return app.Hello() + row }
