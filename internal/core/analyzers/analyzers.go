// Package analyzers holds the architecture checks go-baseline runs on a
// project's code: rules that golangci-lint's linters can't express because
// they need the project's roles and type information across packages.
package analyzers

import (
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

// EnvAdapter is the shared adapter for the process environment, which only a
// composition root may use.
const EnvAdapter = "github.com/ianflanagan1/go-adapters/env"

// Project is what the analyzers need to know about the project under
// analysis.
type Project struct {
	Module string
	Kind   baseline.Kind
	// Impure lists the third-party packages that do I/O, from
	// baseline.ImpurePackages.
	Impure map[string]bool
	// PureModules lists the other modules whose module kind is pure-library.
	// Their packages count as core: what the project passes them is checked
	// as if they were its own core.
	PureModules map[string]bool
}

// All returns every analyzer for p.
func All(p Project) []*analysis.Analyzer {
	return []*analysis.Analyzer{CoreBoundary(p), ConfigRead(p), PortBranch(p)}
}

// role returns the role of the package at importPath, and false when the
// package isn't the project's own.
func (p Project) role(importPath string) (baseline.Role, bool) {
	switch {
	case importPath == p.Module:
		return baseline.RoleOf(p.Kind, ""), true
	case strings.HasPrefix(importPath, p.Module+"/"):
		return baseline.RoleOf(p.Kind, strings.TrimPrefix(importPath, p.Module+"/")), true
	default:
		return 0, false
	}
}

func (p Project) isCore(pkg *types.Package) bool {
	if pkg == nil {
		return false
	}
	if role, own := p.role(pkg.Path()); own {
		return role == baseline.Core
	}
	for module := range p.PureModules {
		if pkg.Path() == module || strings.HasPrefix(pkg.Path(), module+"/") {
			return true
		}
	}
	return false
}

// isPurePackage reports whether code in pkg can't do I/O: the project's
// core, the standard library's pure packages and third-party packages that
// don't reach an I/O package.
func (p Project) isPurePackage(pkg *types.Package) bool {
	if pkg == nil {
		return true
	}
	path := pkg.Path()
	if _, own := p.role(path); own {
		return p.isCore(pkg)
	}
	if isStandard(path) {
		return !baseline.IsIO(path)
	}
	return p.isCore(pkg) || !p.Impure[path]
}

// isStandard reports whether importPath is in the standard library, whose
// paths have no dot in their first element.
func isStandard(importPath string) bool {
	first, _, _ := strings.Cut(importPath, "/")
	return !strings.Contains(first, ".")
}

func isTestFile(pass *analysis.Pass, pos token.Pos) bool {
	return strings.HasSuffix(pass.Fset.Position(pos).Filename, "_test.go")
}
