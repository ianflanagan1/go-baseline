// Package sleep is a conformance fixture: a function in the core sleeps.
package sleep

import "time"

// F waits a second, then returns 1.
func F() int {
	time.Sleep(time.Second)
	return 1
}
