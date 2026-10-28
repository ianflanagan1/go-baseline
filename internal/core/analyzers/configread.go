package analyzers

import (
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

// ConfigRead returns the analyzer that keeps configuration in the
// composition root. Environment variables, os.Args and the command-line
// flags are read only in cmd/, which passes the values in, so the rest of
// the code takes its configuration as plain values. An adapters module is
// exempt: its environment adapter is how a composition root reads them.
func ConfigRead(p Project) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "configread",
		Doc:  "reports configuration read outside a composition root",
		Run: func(pass *analysis.Pass) (any, error) {
			role, own := p.role(pass.Pkg.Path())
			if !own || role == baseline.Root || p.Kind == baseline.Adapters {
				return nil, nil
			}
			for ident, obj := range pass.TypesInfo.Uses {
				if what := configSource(obj); what != "" && !isTestFile(pass, ident.Pos()) {
					pass.Reportf(ident.Pos(), "%s reads configuration; read it in cmd/ and pass the value in", what)
				}
			}
			return nil, nil
		},
	}
}

// configSource names obj when it reads configuration from outside the
// program, or returns "".
func configSource(obj types.Object) string {
	pkg := obj.Pkg()
	if pkg == nil || obj.Parent() != pkg.Scope() {
		return ""
	}
	name := pkg.Name() + "." + obj.Name()
	switch pkg.Path() {
	case "os":
		switch obj.Name() {
		case "Getenv", "LookupEnv", "Environ", "ExpandEnv", "Args":
			return name
		}
	case "flag":
		if _, isType := obj.(*types.TypeName); !isType && obj.Name() != "NewFlagSet" && !isConst(obj) {
			return name
		}
	case EnvAdapter:
		return "the env adapter"
	}
	return ""
}

func isConst(obj types.Object) bool {
	_, ok := obj.(*types.Const)
	return ok
}
