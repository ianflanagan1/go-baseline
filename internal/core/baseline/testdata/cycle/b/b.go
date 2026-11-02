// Package b imports a.
package b

import "example.com/cycle/a"

// B calls a.
func B() { a.A() }
