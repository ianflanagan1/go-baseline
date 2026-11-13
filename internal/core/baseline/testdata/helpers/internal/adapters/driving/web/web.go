// Package web is a driving adapter that imports an orchestrator helper.
package web

import "example.com/helpers/internal/orchestrators/internal/paging"

// Last calls the helper.
func Last() bool { return paging.Last(1, 2) }
