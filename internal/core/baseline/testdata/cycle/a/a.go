// Package a imports b.
package a

import "example.com/cycle/b"

// A calls b.
func A() { b.B() }
