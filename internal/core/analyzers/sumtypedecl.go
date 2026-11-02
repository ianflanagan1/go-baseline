package analyzers

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// SumTypeDecl returns the analyzer that marks every closed family for
// gochecksumtype. An interface with an unexported method is sealed: only
// its own package can implement it, so its variants are known. Its
// declaration carries a //sumtype:decl comment, so gochecksumtype reports
// each type switch on it that leaves out a variant.
func SumTypeDecl(p Project) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "sumtypedecl",
		Doc:  "reports a sealed interface without a //sumtype:decl comment",
		Run: func(pass *analysis.Pass) (any, error) {
			if _, own := p.role(pass.Pkg.Path()); !own {
				return nil, nil
			}
			for _, file := range pass.Files {
				for _, decl := range file.Decls {
					checkTypeDecl(pass, decl)
				}
			}
			return nil, nil
		},
	}
}

func checkTypeDecl(pass *analysis.Pass, decl ast.Decl) {
	gen, ok := decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.TYPE {
		return
	}
	for _, spec := range gen.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		named, ok := pass.TypesInfo.Defs[ts.Name].Type().(*types.Named)
		if !ok || !isSealed(named) || isMarked(ts.Doc) || (len(gen.Specs) == 1 && isMarked(gen.Doc)) {
			continue
		}
		pass.Reportf(ts.Pos(), "%s is sealed, so it's a closed family; add a //sumtype:decl comment so gochecksumtype checks switches on it", ts.Name.Name)
	}
}

func isMarked(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if strings.TrimSpace(c.Text) == "//sumtype:decl" {
			return true
		}
	}
	return false
}
