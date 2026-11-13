// Package timed is a conformance fixture: a driving adapter that keeps to
// its role. It times an invocation with the system stopwatch, a system
// adapter that driving-adapter code may call, and calls an orchestrator.
package timed

import (
	"time"

	"github.com/ianflanagan1/go-shared-code/adapters/driven/stopwatch"

	"example.com/conformance/internal/orchestrators/app"
)

// Handle answers a request, and says how long that took.
func Handle(watch stopwatch.System) (string, time.Duration) {
	elapsed := watch.Start()
	return app.Hello(), elapsed()
}
