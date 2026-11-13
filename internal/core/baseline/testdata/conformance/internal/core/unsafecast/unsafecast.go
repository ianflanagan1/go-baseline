// Package unsafecast is a conformance fixture: a function converts a pointer
// to a pointer of another type through unsafe.Pointer.
package unsafecast

import "unsafe"

// Bits returns f's bits as an integer, without testing anything.
func Bits(f *float64) *uint64 {
	return (*uint64)(unsafe.Pointer(f))
}
