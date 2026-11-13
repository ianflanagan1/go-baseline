// Package enumclock is a conformance fixture: a method on an enum reads the
// current time.
package enumclock

import "time"

// Shift is an enum.
type Shift int

// The shifts.
const (
	Day Shift = iota
	Night
)

// IsNow reports whether s is the shift at the current time.
func (s Shift) IsNow() bool {
	hour := time.Now().Hour()
	return (s == Day) == (hour >= 6 && hour < 18)
}
