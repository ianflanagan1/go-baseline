package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/ianflanagan1/go-baseline/internal/core/analyzers"
	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

func project(kind baseline.Kind) analyzers.Project {
	return analyzers.Project{
		Module:      "example.com/app",
		Kind:        kind,
		Impure:      map[string]bool{"example.com/sdk": true},
		PureModules: map[string]bool{"example.com/ratelib": true},
	}
}

func TestCoreBoundary(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), analyzers.CoreBoundary(project(baseline.Command)),
		"example.com/app/internal/app", "example.com/app/cmd/app")
}

func TestConfigRead(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), analyzers.ConfigRead(project(baseline.Command)),
		"example.com/app/internal/core/rates", "example.com/app/internal/config",
		"example.com/app/internal/adapters/store", "example.com/app/cmd/config")
}

func TestAnAdaptersModuleMayReadConfiguration(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), analyzers.ConfigRead(analyzers.Project{
		Module: "github.com/ianflanagan1/go-adapters", Kind: baseline.Adapters,
	}), "github.com/ianflanagan1/go-adapters/env")
}

func TestAllReturnsEveryAnalyzer(t *testing.T) {
	if got := len(analyzers.All(project(baseline.Command))); got != 2 {
		t.Fatalf("got %d analyzers, want 2", got)
	}
}
