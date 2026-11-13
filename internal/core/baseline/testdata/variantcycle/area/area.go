// Package area references the sealed union whose variant references it.
package area

import "example.com/variantcycle/shape"

// Unit is a unit of area.
type Unit int

// Of returns s's area.
func Of(s shape.Shape) float64 {
	if s == nil {
		return 0
	}
	return 1
}
