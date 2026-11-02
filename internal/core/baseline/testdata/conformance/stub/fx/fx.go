// Package fx stands in for go.uber.org/fx, a dependency-injection container.
package fx

// Option configures an App.
type Option struct{}

// Provide registers constructors.
func Provide(...any) Option { return Option{} }
