// Package loadtime is a conformance fixture.
package loadtime

import "os"

func init() {
	_ = os.Getenv("HOME")
}

// F returns 1.
func F() int { return 1 }
