package baseline_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
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
// generates, over testdata/conformance: a module with a
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
	config, err := baseline.GolangciConfig("example.com/conformance")
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
	for clause, want := range conformanceClauses() {
		t.Run(clause, func(t *testing.T) {
			for _, w := range want {
				if !slices.Contains(issues, w) {
					t.Errorf("%s didn't report %s", w.linter, w.dir)
				}
			}
		})
	}
	t.Run("code that keeps to its role is not reported", func(t *testing.T) {
		compliant := []string{
			"internal/core/pure", "internal/orchestrators/app", "internal/adapters/driven/notes",
			"internal/adapters/driving/server", "internal/adapters/driving/timed",
		}
		claimed := []string{"depguard", "forbidigo", "gochecknoglobals", "funlen", "gocognit", "gosec", "modernize", "revive"}
		for _, i := range issues {
			if slices.Contains(compliant, i.dir) && slices.Contains(claimed, i.linter) {
				t.Errorf("%s reported %s", i.linter, i.dir)
			}
		}
	})
}

// conformanceClauses returns, for each clause, the issues its violations'
// fixtures should get.
func conformanceClauses() map[string][]issue {
	clauses := architectureClauses()
	maps.Copy(clauses, map[string][]issue{
		// Design
		"nothing-is-imported-by-a-computed-path": {
			{"depguard", "internal/core/loadsplugin"}, {"depguard", "internal/orchestrators/loadsplugin"},
		},
		"visibility-is-not-bypassed":                {{"gosec", "internal/core/unsafefield"}},
		"no-cast-asserts-an-unchecked-type":         {{"gosec", "internal/core/unsafecast"}},
		"loops-compute-no-value-an-operation-names": {{"modernize", "internal/core/loopcontains"}},
		"no-positional-groups":                      {{"revive", "internal/core/errorfirst"}},
		"shared-collections-are-not-changed":        {{"revive", "internal/core/appendsparameter"}},
		"closed-families-are-sealed":                {{"gochecksumtype", "internal/core/unsealed"}},
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

		// Errors
		"error-types-of-other-roles-are-not-matched": {
			{"depguard", "internal/adapters/driving/matchesdriven"}, {"depguard", "internal/orchestrators/matchesdriven"},
		},
		"returned-failures-are-used": {
			{"errcheck", "internal/core/droppederror"}, {"ineffassign", "internal/core/unreadsentinel"},
		},
		"failure-branches-act": {{"revive", "internal/core/emptybranch"}},

		// Testing
		"fake-code-performs-only-held-state": {{"gochecknoglobals", "internal/orchestrators/fakeglobal"}},
	})
	return clauses
}

func architectureClauses() map[string][]issue {
	return map[string][]issue{
		"core-references-only-core": {
			{"depguard", "internal/core/importsorchestrator"},
			{"depguard", "internal/core/importsadapter"},
			{"depguard", "internal/core/importsdriving"},
		},
		"core-does-nothing-impure": append(effects("internal/core/", "depguard"), issue{"forbidigo", "internal/core/enumclock"}),
		"orchestrator-code-references-no-entrypoint-or-adapter-code": {
			{"depguard", "internal/orchestrators/importsadapter"}, {"depguard", "internal/orchestrators/importsdriving"},
		},
		"orchestrator-code-performs-no-effects": effects("internal/orchestrators/", "forbidigo"),
		"core-loading-does-nothing-impure":      {{"gochecknoinits", "internal/core/loadtime"}},
		"port-is-declared-by-its-caller":        {{"depguard", "internal/orchestrators/importsadapter"}},
		"port-units-of-code-follow-interface-implementation": {
			{"depguard", "internal/adapters/driven/importsorchestrator"},
		},
		"orchestrator-is-given-ports": {
			{"depguard", "internal/orchestrators/importsadapter"}, {"depguard", "internal/orchestrators/container"},
		},
		"driving-adapter-code-references-only-orchestrators-and-the-core": {
			{"depguard", "internal/adapters/driving/importsdriven"},
		},
		"driven-adapter-code-references-only-ports-and-values": {
			{"depguard", "internal/adapters/driven/importsorchestrator"},
			{"depguard", "internal/adapters/driven/internal/rows"},
			{"depguard", "internal/adapters/driven/importsdriving"},
			{"depguard", "internal/adapters/driven/importsentrypoint"},
		},
		"driving-adapter-code-calls-no-driven-adapter": {{"depguard", "internal/adapters/driving/importsdriven"}},
		"driven-adapter-code-calls-no-orchestrator":    {{"depguard", "internal/adapters/driven/importsorchestrator"}},
		"adapter-code-function-size-is-limited":        {{"funlen", "internal/adapters/driven/long"}},
		"entrypoint-code-function-size-is-limited":     {{"funlen", "cmd/app"}},
		"entrypoint-code-wires-without-a-container":    {{"depguard", "internal/orchestrators/container"}, {"depguard", "cmd/container"}},
		"value-code-references-only-value-code":        {{"depguard", "internal/core/errorreferencesorchestrator"}},
		"adapters-and-entrypoints-perform-only-their-roles-operations": {
			{"gochecknoglobals", "internal/adapters/driven/globalstate"},
		},
		"nothing-sleeps": {{"forbidigo", "internal/core/sleep"}, {"forbidigo", "internal/orchestrators/sleep"}},
		"reserved-operations-are-performed-only-by-their-system-adapters": {
			{"forbidigo", "internal/core/clock"},
			{"depguard", "internal/core/randomness"},
			{"forbidigo", "internal/orchestrators/clock"},
			{"depguard", "internal/orchestrators/randomness"},
			{"gosec", "internal/adapters/driven/pseudorandom"},
		},
	}
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

// buildFails runs `go build ./...` over the fixture module in dir, and
// returns what the compiler said when the build failed.
func buildFails(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("%s built:\n%s", dir, out)
	}
	return string(out)
}

// TestTheCompilerRejectsImportCycles shows the language enforcing
// no-reference-cycles. In testdata/cycle two packages import each other. In
// testdata/variantcycle a sealed union's variant references a package that
// references the union: Go keeps a union and its variants in one package,
// so that is a cycle between packages too.
func TestTheCompilerRejectsImportCycles(t *testing.T) {
	for name, dir := range map[string]string{
		"two packages that import each other":                              "testdata/cycle",
		"a sealed union's variant and a package that references the union": "testdata/variantcycle",
	} {
		t.Run(name, func(t *testing.T) {
			if out := buildFails(t, dir); !strings.Contains(out, "import cycle not allowed") {
				t.Errorf("got:\n%s\nwant an import cycle error", out)
			}
		})
	}
}

// TestTheCompilerKeepsARolesHelpersFromOtherRoles shows the language
// enforcing other-roles-reach-only-members, over testdata/helpers, whose
// driving adapter imports a helper under internal/orchestrators/internal/.
func TestTheCompilerKeepsARolesHelpersFromOtherRoles(t *testing.T) {
	out := buildFails(t, "testdata/helpers")
	if !strings.Contains(out, "use of internal package example.com/helpers/internal/orchestrators/internal/paging not allowed") {
		t.Errorf("got:\n%s\nwant an internal package error", out)
	}
}

// TestTheCompilerLetsNoPackageImportAnEntrypoint shows the language keeping
// every other role's code from referencing an entrypoint, over
// testdata/importsentrypoint, whose orchestrator and adapters each import
// its main package.
func TestTheCompilerLetsNoPackageImportAnEntrypoint(t *testing.T) {
	out := buildFails(t, "testdata/importsentrypoint")
	for _, dir := range []string{"internal/orchestrators/wired", "internal/adapters/driving/wired", "internal/adapters/driven/wired"} {
		want := filepath.FromSlash(dir) + string(filepath.Separator) + `wired.go:4:8: import "example.com/importsentrypoint/cmd/app" is a program, not an importable package`
		if !strings.Contains(out, want) {
			t.Errorf("got:\n%s\nwant %s", out, want)
		}
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
	config, err := baseline.GolangciConfig("example.com/unformatted")
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
		"no-parameter-default-values":      "func f(n int = 1) int { return n }\n",
		"no-optional-parameters":           "func f(n? int) int { return n }\n",
		"field-initialisers-are-constants": "type T struct {\n\tn int = 1\n}\n",
	} {
		t.Run(clause, func(t *testing.T) {
			if _, err := parser.ParseFile(token.NewFileSet(), "f.go", "package p\n\n"+src, 0); err == nil {
				t.Errorf("the parser accepted:\n%s", src)
			}
		})
	}
}

// TestTheCompilerRejects shows the language enforcing the clauses that Go's
// type checker leaves no way to break: the parser or the type checker
// rejects each declaration. Each subtest is named for its clause.
func TestTheCompilerRejects(t *testing.T) {
	for clause, sources := range map[string]map[string]string{
		"parameters-are-typed": {
			"a parameter with no type": "func f(n) int { return 1 }\n",
		},
		"return-types-are-declared-where-called-from-outside": {
			"a function that returns a value and declares no result": "func F() { return 1 }\n",
		},
		"fields-are-typed": {
			"a field with no type": "type T struct{ n }\n",
		},
		"collection-types-name-their-element-type": {
			"a slice with no element type": "func F(items []) int { return len(items) }\n",
			"a map with no value type":     "func F(byID map[string]) int { return len(byID) }\n",
			"a map with no key type":       "func F(byID map[]int) int { return len(byID) }\n",
		},
		"no-type-turns-checking-off": {
			"arithmetic on an any":        "func F(v any) int { return v + 1 }\n",
			"a field read from an any":    "func F(v any) string { return v.name }\n",
			"an any passed as a string":   "func F(v any) int { return len(g(v)) }\n\nfunc g(s string) string { return s }\n",
			"an any assigned to an int":   "func F(v any) int {\n\tvar n int = v\n\treturn n\n}\n",
			"a method called on an any":   "func F(v any) string { return v.Name() }\n",
			"an any called as a function": "func F(v any) { v() }\n",
		},
		"type-checker-runs-at-its-strictest": {
			"a value of the wrong type": "var n int = \"one\"\n",
		},
	} {
		t.Run(clause, func(t *testing.T) {
			for name, src := range sources {
				if compiles("package p\n\n" + src) {
					t.Errorf("%s: the compiler accepted:\n%s", name, src)
				}
			}
		})
	}
}

// compiles reports whether the parser and the type checker both accept src,
// a file that imports nothing.
func compiles(src string) bool {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "f.go", src, 0)
	if err != nil {
		return false
	}
	_, err = (&types.Config{}).Check("example.com/p", fset, []*ast.File{file}, nil)
	return err == nil
}

// TestTheCompilerAcceptsWhatTheCompilerTestsBuildOn guards
// TestTheCompilerRejects against a source it rejects for another reason.
func TestTheCompilerAcceptsWhatTheCompilerTestsBuildOn(t *testing.T) {
	for name, src := range map[string]string{
		"a typed parameter":            "func f(n int) int { return n }\n",
		"a declared result":            "func F() int { return 1 }\n",
		"a typed field":                "type T struct{ n int }\n",
		"a slice of ints":              "func F(items []int) int { return len(items) }\n",
		"an any narrowed with a test":  "func F(v any) int {\n\tn, ok := v.(int)\n\tif !ok {\n\t\treturn 0\n\t}\n\treturn n + 1\n}\n",
		"a value of the declared type": "var n int = 1\n",
	} {
		t.Run(name, func(t *testing.T) {
			if !compiles("package p\n\n" + src) {
				t.Errorf("the compiler rejected:\n%s", src)
			}
		})
	}
}

// TestGoHasNoBuiltinThatRunsText shows the language enforcing
// no-code-is-run-from-text: Go has no eval. Its only functions that need no
// import are the built-ins, and none of them takes code as text. A built-in
// added to the language fails this test until someone has read what it does.
func TestGoHasNoBuiltinThatRunsText(t *testing.T) {
	known := []string{
		"append", "cap", "clear", "close", "complex", "copy", "delete", "imag", "len", "make", "max", "min", "new",
		"panic", "print", "println", "real", "recover",
	}
	for _, name := range types.Universe.Names() {
		if _, builtin := types.Universe.Lookup(name).(*types.Builtin); builtin && !slices.Contains(known, name) {
			t.Errorf("Go has a built-in function this test doesn't know, %s; check that it runs no code from text", name)
		}
	}
	if compiles("package p\n\nfunc F() { eval(\"1 + 1\") }\n") {
		t.Error("the compiler accepted a call to eval")
	}
}

// TestAnInterfaceWithAnUnexportedMethodIsImplementedOnlyInItsPackage shows
// the language enforcing sealed-variants-share-their-unit-of-code and no-port-is-sealed:
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
