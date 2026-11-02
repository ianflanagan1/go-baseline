package analyzers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// NoDefault returns the analyzer that keeps default arms out of switches
// over closed families. A switch on one of the project's enums, or a type
// switch on one of its sealed interfaces, names every case, so adding a
// value or a variant makes exhaustive or gochecksumtype report each switch
// that must handle it. A default arm would take the new case silently.
//
// An enum is a defined type with a basic underlying type that its package
// declares constants of. A sealed interface has an unexported method, so
// only its own package can implement it.
func NoDefault(p Project) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "nodefault",
		Doc:  "reports a default arm in a switch over an enum or a sealed interface",
		Run: func(pass *analysis.Pass) (any, error) {
			if _, own := p.role(pass.Pkg.Path()); !own {
				return nil, nil
			}
			for _, file := range pass.Files {
				ast.Inspect(file, func(n ast.Node) bool {
					checkSwitch(pass, p, n)
					return true
				})
			}
			return nil, nil
		},
	}
}

func checkSwitch(pass *analysis.Pass, p Project, n ast.Node) {
	switch s := n.(type) {
	case *ast.SwitchStmt:
		if s.Tag == nil {
			return
		}
		if named, ok := ownNamed(p, pass.TypesInfo.TypeOf(s.Tag)); ok && isEnum(named) {
			reportDefault(pass, s.Body, named, "value")
		}
	case *ast.TypeSwitchStmt:
		if named, ok := ownNamed(p, pass.TypesInfo.TypeOf(typeSwitchSubject(s))); ok && isSealed(named) {
			reportDefault(pass, s.Body, named, "variant")
		}
	}
}

func reportDefault(pass *analysis.Pass, body *ast.BlockStmt, named *types.Named, what string) {
	for _, stmt := range body.List {
		if clause, ok := stmt.(*ast.CaseClause); ok && clause.List == nil {
			pass.Reportf(clause.Pos(), "the switch on %s has a default arm; name each %s instead, listing those that share a branch in one case",
				named.Obj().Name(), what)
		}
	}
}

// typeSwitchSubject returns x in `switch x.(type)` or `switch v := x.(type)`.
func typeSwitchSubject(s *ast.TypeSwitchStmt) ast.Expr {
	var expr ast.Expr
	switch assign := s.Assign.(type) {
	case *ast.AssignStmt:
		expr = assign.Rhs[0]
	case *ast.ExprStmt:
		expr = assign.X
	}
	if assertion, ok := expr.(*ast.TypeAssertExpr); ok {
		return assertion.X
	}
	return nil
}

// ownNamed returns t as a defined type of the project's own, so a switch on a
// type from another module, whose cases this code may not know, passes.
func ownNamed(p Project, t types.Type) (*types.Named, bool) {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return nil, false
	}
	if _, own := p.role(named.Obj().Pkg().Path()); !own {
		return nil, false
	}
	return named, true
}

func isEnum(named *types.Named) bool {
	if _, basic := named.Underlying().(*types.Basic); !basic {
		return false
	}
	scope := named.Obj().Pkg().Scope()
	for _, name := range scope.Names() {
		if c, ok := scope.Lookup(name).(*types.Const); ok && types.Identical(c.Type(), named) {
			return true
		}
	}
	return false
}

func isSealed(named *types.Named) bool {
	iface, ok := named.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	for method := range iface.Methods() {
		if !method.Exported() {
			return true
		}
	}
	return false
}
