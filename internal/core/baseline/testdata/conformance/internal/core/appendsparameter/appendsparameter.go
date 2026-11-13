// Package appendsparameter is a conformance fixture: a function appends to a
// slice it was passed, writing into the backing array the caller still holds.
package appendsparameter

// WithOne returns items with 1 added.
func WithOne(items []int) []int {
	items = append(items, 1)
	return items
}
