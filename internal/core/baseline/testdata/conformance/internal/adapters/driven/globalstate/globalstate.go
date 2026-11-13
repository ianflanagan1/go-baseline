// Package globalstate is a conformance fixture: a driven adapter keeps its
// rows in a package-level variable.
package globalstate

var rows []string

// Store is a store of names.
type Store struct{}

// Save keeps name.
func (Store) Save(name string) { rows = append(rows, name) }

// Count returns how many names are kept.
func (Store) Count() int { return len(rows) }
