// Package projectmap describes a project's architecture for someone
// reviewing it: each package's role, the ports the shell declares and the
// adapters that implement them.
package projectmap

import (
	"fmt"
	"go/types"
	"slices"
	"strings"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

// Port is an interface the shell declares, with the adapter types found to
// implement it.
type Port struct {
	Name            string
	Implementations []string
}

// Package is one of the project's packages.
type Package struct {
	Path  string
	Role  baseline.Role
	Ports []Port
}

// Describe describes the project's packages, pkgs. Implementations are
// looked for in the project's adapters and in the third-party packages its
// composition roots and adapters import.
func Describe(module string, kind baseline.Kind, pkgs []*types.Package) []Package {
	candidates := adapterTypes(module, kind, pkgs)
	described := make([]Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		d := Package{Path: pkg.Path(), Role: roleOf(module, kind, pkg)}
		if d.Role == baseline.Shell {
			d.Ports = ports(pkg, candidates)
		}
		described = append(described, d)
	}
	slices.SortFunc(described, func(a, b Package) int { return strings.Compare(a.Path, b.Path) })
	return described
}

// Format lays out packages as a table for a terminal.
func Format(kind baseline.Kind, packages []Package) string {
	var rows [][2]string
	for _, p := range packages {
		rows = append(rows, [2]string{p.Path, p.Role.String()})
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
	fmt.Fprintf(&out, "Module kind: %s\n\n", kind)
	for _, row := range rows {
		fmt.Fprintf(&out, "%-*s  %s\n", width, row[0], row[1])
	}
	return out.String()
}

func roleOf(module string, kind baseline.Kind, pkg *types.Package) baseline.Role {
	return baseline.RoleOf(kind, strings.TrimPrefix(strings.TrimPrefix(pkg.Path(), module), "/"))
}

// adapterTypes returns the named types that may implement a port: those in
// the project's adapters, and in the third-party packages that its
// composition roots and adapters import.
func adapterTypes(module string, kind baseline.Kind, pkgs []*types.Package) []*types.TypeName {
	var candidates []*types.TypeName
	seen := map[*types.Package]bool{}
	add := func(pkg *types.Package) {
		if seen[pkg] {
			return
		}
		seen[pkg] = true
		for _, name := range pkg.Scope().Names() {
			if tn, ok := pkg.Scope().Lookup(name).(*types.TypeName); ok && !types.IsInterface(tn.Type()) {
				candidates = append(candidates, tn)
			}
		}
	}
	for _, pkg := range pkgs {
		role := roleOf(module, kind, pkg)
		if role != baseline.Adapter && role != baseline.Root {
			continue
		}
		if role == baseline.Adapter {
			add(pkg)
		}
		for _, imported := range pkg.Imports() {
			if first, _, _ := strings.Cut(imported.Path(), "/"); strings.Contains(first, ".") && !strings.HasPrefix(imported.Path(), module) {
				add(imported)
			}
		}
	}
	return candidates
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
		port := Port{Name: name}
		for _, c := range candidates {
			if types.Implements(c.Type(), iface) || types.Implements(types.NewPointer(c.Type()), iface) {
				port.Implementations = append(port.Implementations, c.Pkg().Name()+"."+c.Name()+" ("+c.Pkg().Path()+")")
			}
		}
		found = append(found, port)
	}
	return found
}
