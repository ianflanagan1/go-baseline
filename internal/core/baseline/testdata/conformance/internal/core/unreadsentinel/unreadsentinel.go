// Package unreadsentinel is a conformance fixture: the result of a function
// that returns a sentinel, assigned to a variable that is never read.
package unreadsentinel

func lookup(m map[string]int, k string) (int, bool) {
	v, ok := m[k]
	return v, ok
}

// Use reads found from the first lookup, but not from the second.
func Use(m map[string]int) int {
	v, found := lookup(m, "a")
	if !found {
		return 0
	}
	v, found = lookup(m, "b")
	return v
}
