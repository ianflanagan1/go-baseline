// Package unsealed is a conformance fixture: a family marked as a sum type
// whose interface any package can implement.
package unsealed

// Result is a success or a failure.
//
//sumtype:decl
type Result interface{ IsResult() }

// Success is a Result.
type Success struct{}

// IsResult marks Success as a Result.
func (Success) IsResult() {}
