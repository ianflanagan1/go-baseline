package baseline_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

// issue is what golangci-lint reports, reduced to the linter and the
// package's directory.
type issue struct {
	linter, dir string
}

// lintConformanceFixture runs golangci-lint, with the config the baseline
// generates for a command module, over testdata/conformance: a module with a
// package for each violation the rulebook's clauses describe.
func lintConformanceFixture(t *testing.T) []issue {
	t.Helper()
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Fatal("golangci-lint isn't on the PATH; run the tests through devbox")
	}
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS("testdata/conformance")); err != nil {
		t.Fatal(err)
	}
	config, err := baseline.GolangciConfig("example.com/conformance", baseline.Command)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".golangci.yml"), config, 0o600); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(t.TempDir(), "report.json")
	cmd := exec.Command("golangci-lint", "run", "--issues-exit-code=0", "--max-issues-per-linter=0",
		"--max-same-issues=0", "--output.text.path=stderr", "--output.json.path="+report, "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("golangci-lint: %v\n%s", err, out)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Issues []struct {
			FromLinter string
			Pos        struct{ Filename string }
		}
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	issues := make([]issue, 0, len(parsed.Issues))
	for _, i := range parsed.Issues {
		issues = append(issues, issue{linter: i.FromLinter, dir: path.Dir(filepath.ToSlash(i.Pos.Filename))})
	}
	return issues
}

// TestConformance shows the generated config catching each violation of
// the rulebook's clauses that it claims, in rulebook-coverage.json. Each
// subtest is named for its clause.
func TestConformance(t *testing.T) {
	issues := lintConformanceFixture(t)
	clauses := map[string][]issue{
		"core-imports-only-core": {
			{"depguard", "internal/core/importsorchestrator"},
			{"depguard", "internal/core/importsadapter"},
			{"depguard", "internal/core/importsdriving"},
		},
		"core-performs-no-effects": effects("internal/core/", "depguard"),
		"orchestrator-imports-no-adapter": {
			{"depguard", "internal/orchestrators/importsadapter"}, {"depguard", "internal/orchestrators/importsdriving"},
		},
		"orchestrator-performs-no-effects": effects("internal/orchestrators/", "forbidigo"),
		"ports-declared-by-their-user":     {{"depguard", "internal/orchestrators/importsadapter"}},
		"port-units-follow-typing":         {{"depguard", "internal/adapters/driven/importsorchestrator"}},
		"driving-adapter-imports-only-core-orchestrators-and-driving-adapters": {
			{"depguard", "internal/adapters/driving/importsdriven"},
		},
		"driven-adapter-imports-only-core-ports-and-driven-adapters": {
			{"depguard", "internal/adapters/driven/importsorchestrator"}, {"depguard", "internal/adapters/driven/importsdriving"},
		},
		"driving-adapter-calls-no-driven-adapter": {{"depguard", "internal/adapters/driving/importsdriven"}},
		"driven-adapter-calls-no-orchestrator":    {{"depguard", "internal/adapters/driven/importsorchestrator"}},
		"adapter-size-is-limited":                 {{"funlen", "internal/adapters/driven/long"}},
		"entrypoint-size-is-limited":              {{"funlen", "cmd/app"}},
		"wired-without-a-container":               {{"depguard", "internal/orchestrators/container"}, {"depguard", "cmd/container"}},
		"closed-families-are-sealed":              {{"gochecksumtype", "internal/core/unsealed"}},
		"matches-on-closed-families-are-exhaustive": {
			{"gochecksumtype", "internal/core/missingvariant"}, {"exhaustive", "internal/core/missingvalue"},
		},
		"matches-on-closed-families-have-no-default-arm": {{"exhaustive", "internal/core/enumdefault"}},
		"parameters-are-not-reassigned": {
			{"revive", "internal/core/assignedparameter"},
			{"revive", "internal/core/compoundparameter"},
			{"revive", "internal/core/incrementedparameter"},
			{"revive", "internal/core/defaultedparameter"},
			{"revive", "internal/core/closureparameter"},
		},
		"returned-failures-are-used": {
			{"errcheck", "internal/core/droppederror"}, {"ineffassign", "internal/core/unreadsentinel"},
		},
	}
	for clause, want := range clauses {
		t.Run(clause, func(t *testing.T) {
			for _, w := range want {
				if !slices.Contains(issues, w) {
					t.Errorf("%s didn't report %s", w.linter, w.dir)
				}
			}
		})
	}
	t.Run("code that keeps to its role is not reported", func(t *testing.T) {
		for _, i := range issues {
			if slices.Contains([]string{"internal/core/pure", "internal/app", "internal/adapters/driven/clock", "internal/adapters/driving/server"}, i.dir) &&
				slices.Contains([]string{"depguard", "forbidigo", "gochecknoglobals", "funlen", "gocognit"}, i.linter) {
				t.Errorf("%s reported %s", i.linter, i.dir)
			}
		}
	})
}

// effects returns the issue each effect's fixture under prefix should get.
// The global logger is the log package, which the core can't import and
// an orchestrator can import but not call.
func effects(prefix, logging string) []issue {
	return []issue{
		{"forbidigo", prefix + "clock"},
		{"depguard", prefix + "randomness"},
		{"depguard", prefix + "filesystem"},
		{"depguard", prefix + "network"},
		{"depguard", prefix + "process"},
		{"forbidigo", prefix + "console"},
		{logging, prefix + "logging"},
		{"depguard", prefix + "configuration"},
		{"gochecknoglobals", prefix + "globalstate"},
	}
}

// TestTheCompilerRejectsImportCycles shows the language enforcing
// no-import-cycles, over testdata/cycle, whose two packages import each other.
func TestTheCompilerRejectsImportCycles(t *testing.T) {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = "testdata/cycle"
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "import cycle not allowed") {
		t.Errorf("got %v:\n%s\nwant an import cycle error", err, out)
	}
}

// TestTheLintTaskFailsOnUnformattedFiles shows `golangci-lint fmt --diff`,
// which the lint task runs, failing over testdata/unformatted, whose file
// is indented with spaces and has its imports out of order.
func TestTheLintTaskFailsOnUnformattedFiles(t *testing.T) {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Fatal("golangci-lint isn't on the PATH; run the tests through devbox")
	}
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS("testdata/unformatted")); err != nil {
		t.Fatal(err)
	}
	config, err := baseline.GolangciConfig("example.com/unformatted", baseline.PureLibrary)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".golangci.yml"), config, 0o600); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(baseline.Taskfile()), "golangci-lint fmt --diff") {
		t.Fatal("the lint task no longer runs golangci-lint fmt --diff")
	}
	cmd := exec.Command("golangci-lint", "fmt", "--diff")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("golangci-lint fmt --diff passed:\n%s", out)
	}
	for _, want := range []string{"-    return", "+\treturn", "+\t\"strings\""} {
		if !strings.Contains(string(out), want) {
			t.Errorf("the diff doesn't have %q:\n%s", want, out)
		}
	}
}

// TestGoHasNoSyntaxFor shows the language enforcing the clauses about
// constructs Go doesn't have: the parser rejects the syntax other languages
// use for them. Each subtest is named for its clause.
func TestGoHasNoSyntaxFor(t *testing.T) {
	for clause, src := range map[string]string{
		"classes-extend-only-allowed-bases": "type Base struct{}\n\ntype Derived extends Base\n",
		"no-parameter-default-values":       "func f(n int = 1) int { return n }\n",
		"no-optional-parameters":            "func f(n? int) int { return n }\n",
		"field-initialisers-are-constants":  "type T struct {\n\tn int = 1\n}\n",
	} {
		t.Run(clause, func(t *testing.T) {
			if _, err := parser.ParseFile(token.NewFileSet(), "f.go", "package p\n\n"+src, 0); err == nil {
				t.Errorf("the parser accepted:\n%s", src)
			}
		})
	}
}

// TestAnInterfaceWithAnUnexportedMethodIsImplementedOnlyInItsPackage shows
// the language enforcing sealed-variants-share-their-unit and ports-are-open:
// a type in another package can't implement an interface with an unexported
// method, so a sealed union's variants are all in its package, and a port
// sealed that way can't be implemented by the adapters and fakes outside it.
func TestAnInterfaceWithAnUnexportedMethodIsImplementedOnlyInItsPackage(t *testing.T) {
	fset := token.NewFileSet()
	parse := func(src string) *ast.File {
		f, err := parser.ParseFile(fset, "f.go", src, 0)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	sealed, err := (&types.Config{}).Check("example.com/sealed", fset,
		[]*ast.File{parse("package sealed\n\ntype Step interface{ isStep() }\n")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	outside := parse("package outside\n\nimport \"example.com/sealed\"\n\ntype Retry struct{}\n\nfunc (Retry) isStep() {}\n\nvar _ sealed.Step = Retry{}\n")
	importer := importerFunc(func(string) (*types.Package, error) { return sealed, nil })
	_, err = (&types.Config{Importer: importer}).Check("example.com/outside", fset, []*ast.File{outside}, nil)
	if err == nil || !strings.Contains(err.Error(), "does not implement sealed.Step (unexported method isStep)") {
		t.Errorf("got %v, want the type checker to reject the implementation", err)
	}
}

type importerFunc func(path string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }
