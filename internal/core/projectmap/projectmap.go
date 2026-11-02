// Package projectmap describes a project's architecture for someone
// reviewing it: each package's role, the ports the orchestrators declare and
// the driven adapters that implement them.
package projectmap

import (
	"fmt"
	"go/types"
	"slices"
	"strings"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

// Port is an interface an orchestrator declares, with the driven adapter
// types found to implement it.
type Port struct {
	Name            string
	Implementations []string
}

// Package is one of the project's packages.
type Package struct {
	Path string
	Role baseline.Role
	// HasRole is false for a package whose directory gives it no role.
	HasRole bool
	Ports   []Port
}

// Describe describes the project's packages, pkgs. Implementations are
// looked for in the project's driven adapters and in the third-party packages
// its entrypoints and driven adapters import.
func Describe(module string, pkgs []*types.Package) []Package {
	candidates := adapterTypes(module, pkgs)
	described := make([]Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		role, ok := roleOf(module, pkg)
		d := Package{Path: pkg.Path(), Role: role, HasRole: ok}
		if ok && role == baseline.Orchestrator {
			d.Ports = ports(pkg, candidates)
		}
		described = append(described, d)
	}
	slices.SortFunc(described, func(a, b Package) int { return strings.Compare(a.Path, b.Path) })
	return described
}

// Format lays out packages as a table for a terminal.
func Format(packages []Package) string {
	var rows [][2]string
	for _, p := range packages {
		role := "no role: move it into a role's directory"
		if p.HasRole {
			role = p.Role.String()
		}
		rows = append(rows, [2]string{p.Path, role})
		for _, port := range p.Ports {
			implementations := "no implementation found"
			if len(port.Implementations) > 0 {
				implementations = strings.Join(port.Implementations, ", ")
			}
			rows = append(rows, [2]string{"  port " + port.Name, implementations})
		}
	}
	width := 0
	for _, row := range rows {
		width = max(width, len(row[0]))
	}
	var out strings.Builder
	for _, row := range rows {
		fmt.Fprintf(&out, "%-*s  %s\n", width, row[0], row[1])
	}
	return out.String()
}

func roleOf(module string, pkg *types.Package) (baseline.Role, bool) {
	return baseline.RoleOf(strings.TrimPrefix(strings.TrimPrefix(pkg.Path(), module), "/"))
}

// adapterTypes returns the named types that may implement a port: those in
// the project's driven adapters, and in the third-party packages that its
// entrypoints and driven adapters import.
func adapterTypes(module string, pkgs []*types.Package) []*types.TypeName {
	var sources []*types.Package
	for _, pkg := range pkgs {
		role, ok := roleOf(module, pkg)
		if !ok {
			continue
		}
		switch role {
		case baseline.DrivenAdapter:
			sources = append(sources, pkg)
			sources = append(sources, thirdPartyImports(module, pkg)...)
		case baseline.Entrypoint:
			sources = append(sources, thirdPartyImports(module, pkg)...)
		case baseline.Core, baseline.Orchestrator, baseline.DrivingAdapter:
		}
	}
	var candidates []*types.TypeName
	seen := map[*types.Package]bool{}
	for _, pkg := range sources {
		if !seen[pkg] {
			seen[pkg] = true
			candidates = append(candidates, concreteTypes(pkg)...)
		}
	}
	return candidates
}

// thirdPartyImports returns the packages pkg imports from other modules,
// leaving out the standard library.
func thirdPartyImports(module string, pkg *types.Package) []*types.Package {
	var found []*types.Package
	for _, imported := range pkg.Imports() {
		if first, _, _ := strings.Cut(imported.Path(), "/"); strings.Contains(first, ".") && !strings.HasPrefix(imported.Path(), module) {
			found = append(found, imported)
		}
	}
	return found
}

// concreteTypes returns pkg's named types that aren't interfaces.
func concreteTypes(pkg *types.Package) []*types.TypeName {
	var found []*types.TypeName
	for _, name := range pkg.Scope().Names() {
		if tn, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok && !types.IsInterface(tn.Type()) {
			found = append(found, tn)
		}
	}
	return found
}

func ports(pkg *types.Package, candidates []*types.TypeName) []Port {
	var found []Port
	for _, name := range pkg.Scope().Names() {
		tn, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		iface, ok := tn.Type().Underlying().(*types.Interface)
		if !ok || iface.NumMethods() == 0 {
			continue
		}
		found = append(found, Port{Name: name, Implementations: implementations(iface, candidates)})
	}
	return found
}

// implementations names the candidates that implement iface, directly or
// through a pointer.
func implementations(iface *types.Interface, candidates []*types.TypeName) []string {
	var found []string
	for _, c := range candidates {
		if types.Implements(c.Type(), iface) || types.Implements(types.NewPointer(c.Type()), iface) {
			found = append(found, c.Pkg().Name()+"."+c.Name()+" ("+c.Pkg().Path()+")")
		}
	}
	return found
}
