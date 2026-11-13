// Package loopcontains is a conformance fixture: a loop visits each element
// of a list only to say whether one of them equals a value, which
// slices.Contains gives.
package loopcontains

// Has reports whether id is one of ids.
func Has(ids []string, id string) bool {
	for _, each := range ids {
		if each == id {
			return true
		}
	}
	return false
}
