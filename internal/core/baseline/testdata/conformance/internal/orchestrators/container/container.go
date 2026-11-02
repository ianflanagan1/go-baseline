// Package container is a conformance fixture: an orchestrator builds its object
// graph with a container.
package container

import "go.uber.org/fx"

// Options provides the orchestrator's constructors.
func Options() fx.Option { return fx.Provide() }
