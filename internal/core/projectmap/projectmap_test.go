package projectmap_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
	"github.com/ianflanagan1/go-baseline/internal/core/projectmap"
)

// sources returns a command, in dependency order.
func sources() []struct{ path, src string } {
	return []struct{ path, src string }{
		{"example.com/sdk", "package sdk\n\ntype Mailer struct{}\n\nfunc (*Mailer) Notify(string) error { return nil }\n"},
		{"example.com/app/internal/app", "package app\n\ntype Clock interface{ Now() int }\n\ntype Notifier interface{ Notify(string) error }\n\ntype Store interface{ Save() }\n\ntype Empty interface{}\n\ntype Report struct{}\n"},
		{"example.com/app/internal/adapters/driven/clock", "package clock\n\ntype System struct{}\n\nfunc (System) Now() int { return 0 }\n"},
		{"example.com/app/cmd/app", "package main\n\nimport (\n\t\"example.com/app/internal/adapters/driven/clock\"\n\t\"example.com/app/internal/app\"\n\t\"example.com/sdk\"\n)\n\nvar _ app.Clock = clock.System{}\n\nvar _ = &sdk.Mailer{}\n"},
	}
}

type importer map[string]*types.Package

func (i importer) Import(path string) (*types.Package, error) { return i[path], nil }

func check(t *testing.T) []*types.Package {
	t.Helper()
	checked := importer{}
	var own []*types.Package
	for _, s := range sources() {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, s.path+".go", s.src, 0)
		if err != nil {
			t.Fatal(err)
		}
		pkg, err := (&types.Config{Importer: checked}).Check(s.path, fset, []*ast.File{file}, nil)
		if err != nil {
			t.Fatal(err)
		}
		checked[s.path] = pkg
		if strings.HasPrefix(s.path, "example.com/app") {
			own = append(own, pkg)
		}
	}
	return own
}

func TestPortsAreMatchedToAdaptersAndThirdPartyTypes(t *testing.T) {
	described := projectmap.Describe("example.com/app", check(t))
	want := []projectmap.Package{
		{Path: "example.com/app/cmd/app", Role: baseline.Entrypoint, HasRole: true},
		{Path: "example.com/app/internal/adapters/driven/clock", Role: baseline.DrivenAdapter, HasRole: true},
		{Path: "example.com/app/internal/app", Role: baseline.Orchestrator, HasRole: true, Ports: []projectmap.Port{
			{Name: "Clock", Implementations: []string{"clock.System (example.com/app/internal/adapters/driven/clock)"}},
			{Name: "Notifier", Implementations: []string{"sdk.Mailer (example.com/sdk)"}},
			{Name: "Store"},
		}},
	}
	if got, wantText := projectmap.Format(described), projectmap.Format(want); got != wantText {
		t.Fatalf("got\n%s\nwant\n%s", got, wantText)
	}
}

func TestTheFormatShowsRolesAndMissingImplementations(t *testing.T) {
	got := projectmap.Format(projectmap.Describe("example.com/app", check(t)))
	for _, want := range []string{"example.com/app/cmd/app", "entrypoint", "port Store", "no implementation found"} {
		if !strings.Contains(got, want) {
			t.Errorf("got\n%s\nwant it to contain %q", got, want)
		}
	}
}

func TestAPackageWithNoRoleIsShownAsOne(t *testing.T) {
	got := projectmap.Format([]projectmap.Package{{Path: "example.com/app/internal/adapters/clock"}})
	if !strings.Contains(got, "example.com/app/internal/adapters/clock  no role: move it to adapters/driving/ or adapters/driven/") {
		t.Errorf("got\n%s", got)
	}
}
