// Package missingvariant is a conformance fixture: a type switch on a sealed
// union that leaves out a variant.
package missingvariant

// Shape is a Circle or a Square.
//
//sumtype:decl
type Shape interface{ isShape() }

// Circle is a Shape.
type Circle struct{}

// Square is a Shape.
type Square struct{}

func (Circle) isShape() {}
func (Square) isShape() {}

// Name names s.
func Name(s Shape) string {
	switch s.(type) {
	case Circle:
		return "circle"
	}
	return ""
}
