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
	if got := len(analyzers.All(project(baseline.Command))); got != 3 {
		t.Fatalf("got %d analyzers, want 3", got)
	}
}

func TestPortBranch(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), analyzers.PortBranch(project(baseline.Command)),
		"example.com/app/internal/orders", "example.com/app/internal/notify",
		"example.com/app/internal/core/pricing")
}

// ignoreExpectations is for running an analyzer to read its diagnostics
// without // want comments, which every analyzer run on a package must meet.
type ignoreExpectations struct{}

func (ignoreExpectations) Errorf(string, ...any) {}

func TestTheGuidesExamplesPassEveryAnalyzer(t *testing.T) {
	for _, a := range analyzers.All(project(baseline.Command)) {
		t.Run(a.Name, func(t *testing.T) {
			results := analysistest.Run(ignoreExpectations{}, analysistest.TestData(), a, "example.com/app/internal/guide")
			if len(results) == 0 {
				t.Fatal("the guide's examples didn't load")
			}
			for _, result := range results {
				if result.Err != nil {
					t.Fatal(result.Err)
				}
				for _, d := range result.Diagnostics {
					t.Errorf("%s: %s", result.Pass.Fset.Position(d.Pos), d.Message)
				}
			}
		})
	}
}
