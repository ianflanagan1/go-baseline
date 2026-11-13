// Package shape declares a sealed union, one of whose variants references a
// package that references the union.
package shape

import "example.com/variantcycle/area"

// Shape is a circle or a square.
//
//sumtype:decl
type Shape interface{ isShape() }

// Circle is a circle.
type Circle struct{ radius float64 }

// Square is a square, which holds its area's unit.
type Square struct {
	side float64
	unit area.Unit
}

func (Circle) isShape() {}
func (Square) isShape() {}
