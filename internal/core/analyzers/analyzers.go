// Package analyzers holds the checks go-baseline runs on a project's code:
// rules that golangci-lint's linters can't express because they need the
// project's roles, or which of its types are its own.
package analyzers

import (
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

// EnvAdapter is the shared adapter for the process environment, which only an
// entrypoint may use.
const EnvAdapter = "github.com/ianflanagan1/go-shared-code/adapters/driven/env"

// Project is what the analyzers need to know about the project under
// analysis.
type Project struct {
	Module string
	// Impure lists the third-party packages that do I/O, from
	// baseline.ImpurePackages.
	Impure map[string]bool
	// BaselineModules lists the other modules on go-baseline. Their
	// directories give their packages roles, as the project's own do, and
	// their core counts as core: what the project passes it is checked as if
	// it were the project's own core.
	BaselineModules map[string]bool
}

// All returns every analyzer for p.
func All(p Project) []*analysis.Analyzer {
	return []*analysis.Analyzer{CoreBoundary(p), ConfigRead(p), PortBranch(p), NoDefault(p), SumTypeDecl(p)}
}

// role returns the role of the package at importPath, and false when the
// package isn't the project's own or has no role, which verify reports.
func (p Project) role(importPath string) (baseline.Role, bool) {
	switch {
	case importPath == p.Module:
		return baseline.RoleOf("")
	case strings.HasPrefix(importPath, p.Module+"/"):
		return baseline.RoleOf(strings.TrimPrefix(importPath, p.Module+"/"))
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
	for module := range p.BaselineModules {
		if dir, ok := strings.CutPrefix(pkg.Path(), module+"/"); ok {
			role, _ := baseline.RoleOf(dir)
			return role == baseline.Core
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
