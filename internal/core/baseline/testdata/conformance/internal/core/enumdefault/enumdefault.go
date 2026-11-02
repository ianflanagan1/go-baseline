// Package enumdefault is a conformance fixture: a switch on an enum that names
// two of its values and sends the rest to a default arm.
package enumdefault

// Colour is an enum.
type Colour int

// The colours.
const (
	Red Colour = iota
	Green
	Blue
)

// Name names c.
func Name(c Colour) string {
	switch c {
	case Red:
		return "red"
	case Green:
		return "green"
	default:
		return "other"
	}
}
