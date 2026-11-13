// Package notes is a conformance fixture: a driven adapter that keeps to its
// role, reading a file.
package notes

import (
	"fmt"
	"os"
)

// Store reads notes from files.
type Store struct{}

// Read returns the note in the file at path.
func (Store) Read(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return string(data), nil
}
