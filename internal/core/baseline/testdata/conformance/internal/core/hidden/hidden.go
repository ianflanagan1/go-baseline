// Package hidden is a conformance fixture: a core type whose field other
// packages can't see, for other fixtures to reach around.
package hidden

// Total is an amount of money.
type Total struct{ pence int64 }

// NewTotal returns the total of pence.
func NewTotal(pence int64) Total { return Total{pence: pence} }

// Pence returns the total in pence.
func (t Total) Pence() int64 { return t.pence }
