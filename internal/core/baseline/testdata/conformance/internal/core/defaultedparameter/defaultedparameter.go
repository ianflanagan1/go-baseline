// Package defaultedparameter is a conformance fixture.
package defaultedparameter

// Greeter greets.
type Greeter struct{}

// Greet replaces its parameter with a default when it is empty.
func (Greeter) Greet(name string) string {
	if name == "" {
		name = "you"
	}
	return "Hello, " + name
}
