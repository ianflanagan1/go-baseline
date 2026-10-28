package gopackages_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ianflanagan1/go-baseline/internal/adapters/gopackages"
	"github.com/ianflanagan1/go-baseline/internal/core/analyzers"
	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

// module writes a module to a temporary directory, from file names and
// contents.
func module(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files["go.mod"] = "module example.com/t\n\ngo 1.27\n"
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDiagnosticsAreReportedRelativeToTheModule(t *testing.T) {
	dir := module(t, map[string]string{
		"internal/app/app.go": "package app\n\nimport \"os\"\n\nfunc Token() string { return os.Getenv(\"TOKEN\") }\n",
	})
	project := analyzers.Project{Module: "example.com/t", Kind: baseline.Command}
	got, err := gopackages.Analyze(dir, analyzers.All(project))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.HasPrefix(got[0], "internal/app/app.go:5:33: os.Getenv reads configuration") {
		t.Fatalf("got %q", got)
	}
}

func TestCodeThatDoesntCompileIsAnError(t *testing.T) {
	dir := module(t, map[string]string{"app.go": "package app\n\nfunc broken() { undefined() }\n"})
	if _, err := gopackages.Analyze(dir, nil); err == nil {
		t.Fatal("got no error")
	}
}

func TestTypesDescribesEachPackageWithEverythingItImports(t *testing.T) {
	dir := module(t, map[string]string{
		"a/a.go":   "package a\n\n// T is a type.\ntype T int\n",
		"cmd/m.go": "package main\n\nimport \"example.com/t/a\"\n\nfunc main() { _ = a.T(0) }\n",
	})
	got, err := gopackages.Types(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Path() != "example.com/t/a" || got[0].Scope().Lookup("T") == nil {
		t.Fatalf("got %v", got)
	}
	if imports := got[1].Imports(); len(imports) != 1 || imports[0].Path() != "example.com/t/a" {
		t.Fatalf("main imports %v, want example.com/t/a", imports)
	}
}
