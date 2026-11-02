// Package missingvalue is a conformance fixture: a switch on an enum that
// leaves out a value.
package missingvalue

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
	}
	return ""
}
