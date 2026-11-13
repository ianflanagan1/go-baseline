// Package unsafefield is a conformance fixture: a function writes an
// unexported field of another package's struct through reflect and unsafe.
package unsafefield

import (
	"reflect"
	"unsafe"

	"example.com/conformance/internal/core/hidden"
)

// Zero sets total's pence to nothing, past its factory function.
func Zero(total *hidden.Total) {
	field := reflect.ValueOf(total).Elem().FieldByName("pence")
	*(*int64)(unsafe.Pointer(field.UnsafeAddr())) = 0
}
