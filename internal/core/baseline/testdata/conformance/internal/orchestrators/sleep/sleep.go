// Package sleep is a conformance fixture: an orchestrator sleeps between
// attempts.
package sleep

import "time"

// Retry calls attempt twice, a second apart.
func Retry(attempt func()) {
	attempt()
	time.Sleep(time.Second)
	attempt()
}
