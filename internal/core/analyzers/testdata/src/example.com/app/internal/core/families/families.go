// Package families holds switches over closed families, with and without
// default arms.
package families

import "go/ast"

type Colour int

const (
	Red Colour = iota
	Green
	Blue
)

// Shape is sealed: only this package can implement it.
type Shape interface{ isShape() }

type Circle struct{}

type Square struct{}

func (Circle) isShape() {}
func (Square) isShape() {}

// Open is an interface any package can implement.
type Open interface{ Area() float64 }

// Count is a defined type with no constants, so not an enum.
type Count int

func EnumWithDefault(c Colour) string {
	switch c {
	case Red, Green:
		return "warm or green"
	default: // want `the switch on Colour has a default arm; name each value instead`
		return "other"
	}
}

func EnumNamingEveryValue(c Colour) string {
	switch c {
	case Red, Green:
		return "warm or green"
	case Blue:
		return "blue"
	}
	return ""
}

func SealedWithDefault(s Shape) string {
	switch s.(type) {
	case Circle:
		return "circle"
	case Square:
		return "square"
	default: // want `the switch on Shape has a default arm; name each variant instead`
		return ""
	}
}

func SealedBoundWithDefault(s Shape) string {
	switch v := s.(type) {
	case Circle:
		_ = v
		return "circle"
	default: // want `the switch on Shape has a default arm`
		return ""
	}
}

func SealedNamingEveryVariant(s Shape) string {
	switch s.(type) {
	case Circle, Square:
		return "shape"
	}
	return ""
}

func OpenWithDefault(o Open) string {
	switch o.(type) {
	case nil:
		return "none"
	default:
		return "some"
	}
}

func NotAnEnum(n Count) string {
	switch n {
	case 0:
		return "none"
	default:
		return "some"
	}
}

func AnotherModulesFamily(n ast.Node) string {
	switch n.(type) {
	case *ast.Ident:
		return "ident"
	default:
		return "other"
	}
}

func Untagged(c Colour) string {
	switch {
	case c == Red:
		return "red"
	default:
		return "other"
	}
}
