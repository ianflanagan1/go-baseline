// Command container is a conformance fixture: an entrypoint that wires
// with a container.
package main

import "go.uber.org/fx"

func main() { _ = fx.Provide() }
