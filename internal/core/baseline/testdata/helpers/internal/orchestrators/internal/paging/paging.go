// Package paging is an orchestrator helper.
package paging

// Last reports whether a page of size items is the last.
func Last(size, limit int) bool { return size < limit }
