// Package gopackages loads a project's packages with go/packages, and runs
// analyzers over them.
package gopackages

import (
	"errors"
	"fmt"
	"go/types"
	"path/filepath"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

// Analyze runs analyzers over the packages in dir and returns each
// diagnostic as "file:line:col: message (analyzer)", with file relative to
// dir, sorted.
func Analyze(dir string, analyzers []*analysis.Analyzer) ([]string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("finding %s: %w", dir, err)
	}
	pkgs, err := load(dir, packages.LoadAllSyntax)
	if err != nil {
		return nil, err
	}
	graph, err := checker.Analyze(analyzers, pkgs, nil)
	if err != nil {
		return nil, fmt.Errorf("analysing: %w", err)
	}
	var diagnostics []string
	for _, act := range graph.Roots {
		if act.Err != nil {
			return nil, fmt.Errorf("%s on %s: %w", act.Analyzer.Name, act.Package.PkgPath, act.Err)
		}
		diagnostics = append(diagnostics, format(dir, act)...)
	}
	slices.Sort(diagnostics)
	return diagnostics, nil
}

// format returns act's diagnostics as "file:line:col: message (analyzer)",
// with file relative to dir.
func format(dir string, act *checker.Action) []string {
	formatted := make([]string, 0, len(act.Diagnostics))
	for _, d := range act.Diagnostics {
		pos := act.Package.Fset.Position(d.Pos)
		if rel, err := filepath.Rel(dir, pos.Filename); err == nil {
			pos.Filename = rel
		}
		formatted = append(formatted, fmt.Sprintf("%s: %s (%s)", pos, d.Message, act.Analyzer.Name))
	}
	return formatted
}

// Types returns the type information of the packages in dir. They're type
// checked from source, so each one's Imports are complete.
func Types(dir string) ([]*types.Package, error) {
	pkgs, err := load(dir, packages.LoadSyntax)
	if err != nil {
		return nil, err
	}
	typed := make([]*types.Package, 0, len(pkgs))
	for _, p := range pkgs {
		typed = append(typed, p.Types)
	}
	return typed, nil
}

func load(dir string, mode packages.LoadMode) ([]*packages.Package, error) {
	pkgs, err := packages.Load(&packages.Config{Mode: mode, Dir: dir}, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}
	var errs []error
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			errs = append(errs, e)
		}
	})
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}
	return pkgs, nil
}
