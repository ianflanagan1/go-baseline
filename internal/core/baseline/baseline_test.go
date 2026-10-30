package baseline_test

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

const goMod = "module example.com/app\n\ngo 1.27\n"

const devbox = `{
  "packages": ["go@1.27.3", "golangci-lint@` + baseline.GolangciLintVersion + `", "go-task@` + baseline.TaskVersion + `"],
  "env": {"GOTOOLCHAIN": "local"}
}`

func matchingProject(t *testing.T) baseline.Project {
	t.Helper()
	golangci, err := baseline.GolangciConfig("example.com/app", baseline.Command)
	if err != nil {
		t.Fatal(err)
	}
	return baseline.Project{
		GoMod:    []byte(goMod),
		Golangci: golangci,
		Guide:    baseline.Guide(),
		Workflow: baseline.Workflow(),
		Taskfile: baseline.Taskfile(),
		Devbox:   []byte(devbox),
		Dirs:     []string{"cmd", "cmd/app", "internal", "internal/core", "internal/core/greet"},
	}
}

func verify(t *testing.T, p baseline.Project) []string {
	t.Helper()
	problems, err := baseline.Verify(p)
	if err != nil {
		t.Fatal(err)
	}
	return problems
}

func TestModulePathIsReadFromTheModuleLine(t *testing.T) {
	for _, tc := range []struct{ name, goMod, want string }{
		{"bare", "module example.com/app\n", "example.com/app"},
		{"quoted", "module \"example.com/app\"\n", "example.com/app"},
		{"with a comment", "// Deprecated: use v2.\nmodule example.com/app // the app\n", "example.com/app"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := baseline.ModulePath([]byte(tc.goMod))
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestAGoModWithoutAModuleLineIsAnError(t *testing.T) {
	if _, err := baseline.ModulePath([]byte("go 1.27\n")); !errors.Is(err, baseline.ErrNoModule) {
		t.Fatalf("got %v, want ErrNoModule", err)
	}
}

func TestTheGeneratedConfigBansAdaptersFromTheShellByModulePath(t *testing.T) {
	config, err := baseline.GolangciConfig("example.com/app", baseline.Command)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "- pkg: example.com/app/internal/adapters\n") {
		t.Fatalf("config doesn't deny the module's adapters:\n%s", config)
	}
}

func TestTheGeneratedConfigBansAdaptersFromTheShellForAnyModule(t *testing.T) {
	config, err := baseline.GolangciConfig("example.com/lib", baseline.Command)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "- pkg: example.com/lib/internal/adapters\n") {
		t.Fatalf("config doesn't deny the module's adapters:\n%s", config)
	}
}

func TestTheGeneratedConfigRequiresWrappingOnlyOtherModulesErrors(t *testing.T) {
	for _, kind := range []baseline.Kind{baseline.Command, baseline.PureLibrary, baseline.Adapters} {
		config, err := baseline.GolangciConfig("example.com/app", kind)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(config), "ignore-package-globs:\n        - example.com/app/*\n") {
			t.Fatalf("%s config doesn't exempt the module's own errors from wrapcheck:\n%s", kind, config)
		}
	}
}

func TestTheGeneratedConfigLimitsTheComplexityOfEveryPackageButTheCompositionRoot(t *testing.T) {
	cmdExemption := "      - path: (^|/)cmd/\n        linters:\n          - gocognit\n"
	for _, tc := range []struct {
		kind baseline.Kind
		// Besides enabling gocognit and exempting tests.
		exemptions int
	}{
		{baseline.Command, 1},
		{baseline.PureLibrary, 0},
		{baseline.Adapters, 0},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			config, err := baseline.GolangciConfig("example.com/app", tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(string(config), "- gocognit\n") - 2; got != tc.exemptions {
				t.Fatalf("got %d gocognit exemptions besides tests, want %d:\n%s", got, tc.exemptions, config)
			}
			if tc.exemptions == 1 && !strings.Contains(string(config), cmdExemption) {
				t.Fatalf("config doesn't exempt only the composition root:\n%s", config)
			}
		})
	}
}

func TestAProjectOnTheBaselineHasNoProblems(t *testing.T) {
	if problems := verify(t, matchingProject(t)); problems != nil {
		t.Fatalf("got %q", problems)
	}
}

func TestAnEditedConfigIsReported(t *testing.T) {
	p := matchingProject(t)
	p.Golangci = append(p.Golangci, "# local change\n"...)
	assertOneProblem(t, verify(t, p), ".golangci.yml differs")
}

func TestAMissingConfigIsReported(t *testing.T) {
	p := matchingProject(t)
	p.Golangci = nil
	assertOneProblem(t, verify(t, p), ".golangci.yml is missing")
}

func TestAConfigWithoutAKindIsReported(t *testing.T) {
	p := matchingProject(t)
	p.Golangci = []byte("version: \"2\"\n")
	assertOneProblem(t, verify(t, p), "kind")
}

func TestTheConfigRecordsItsKind(t *testing.T) {
	for _, kind := range []baseline.Kind{baseline.Command, baseline.PureLibrary, baseline.Adapters} {
		t.Run(string(kind), func(t *testing.T) {
			config, err := baseline.GolangciConfig("example.com/app", kind)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := baseline.KindOf(config); err != nil || got != kind {
				t.Fatalf("got %q, %v; want %q", got, err, kind)
			}
		})
	}
}

func TestAnUnknownKindIsAnError(t *testing.T) {
	if _, err := baseline.ParseKind("library"); err == nil {
		t.Fatal("got no error")
	}
}

func TestAPureLibraryConfigHasNoShell(t *testing.T) {
	config, err := baseline.GolangciConfig("example.com/lib", baseline.PureLibrary)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(config), "shell:") || !strings.Contains(string(config), "- pkg: context\n") {
		t.Fatalf("want only core rules, including no context:\n%s", config)
	}
}

func TestAConfigForAnotherModuleIsReported(t *testing.T) {
	p := matchingProject(t)
	p.GoMod = []byte("module example.com/renamed\n\ngo 1.27\n")
	assertOneProblem(t, verify(t, p), ".golangci.yml differs")
}

func TestAnEditedGuideIsReported(t *testing.T) {
	p := matchingProject(t)
	p.Guide = append(p.Guide, "Local notes.\n"...)
	assertOneProblem(t, verify(t, p), "BASELINE.md differs")
}

func TestWorkflowDriftIsReported(t *testing.T) {
	p := matchingProject(t)
	p.Workflow = nil
	assertOneProblem(t, verify(t, p), "check.yml is missing")
	p.Workflow = append(baseline.Workflow(), "# local change\n"...)
	assertOneProblem(t, verify(t, p), "check.yml differs")
}

func TestTaskfileDriftIsReported(t *testing.T) {
	p := matchingProject(t)
	p.Taskfile = nil
	assertOneProblem(t, verify(t, p), "Taskfile.yml is missing")
	p.Taskfile = append(baseline.Taskfile(), "  build:\n    cmds: [go build ./...]\n"...)
	assertOneProblem(t, verify(t, p), "put the project's own tasks in Taskfile.project.yml")
}

func TestAMissingGuideIsReported(t *testing.T) {
	p := matchingProject(t)
	p.Guide = nil
	assertOneProblem(t, verify(t, p), "BASELINE.md is missing")
}

func TestALibraryLayoutHasNoProblems(t *testing.T) {
	p := matchingProject(t)
	p.Dirs = []string{"adapters", "adapters/clock", "internal", "internal/core", "internal/core/rates"}
	if problems := verify(t, p); problems != nil {
		t.Fatalf("got %q", problems)
	}
}

func TestDevboxDriftIsReported(t *testing.T) {
	for _, tc := range []struct{ name, from, to, want string }{
		{"go not pinned", `"go@1.27.3"`, `"go@latest"`, "pin Go"},
		{"go pinned to a minor version", `"go@1.27.3"`, `"go@1.27"`, "pin Go"},
		{"another golangci-lint", "golangci-lint@" + baseline.GolangciLintVersion, "golangci-lint@2.0.0", "golangci-lint@"},
		{"toolchain downloads allowed", `"GOTOOLCHAIN": "local"`, `"GOTOOLCHAIN": "auto"`, "GOTOOLCHAIN"},
		{"another go-task", "go-task@" + baseline.TaskVersion, "go-task@3.0.0", "go-task@"},
		{"a devbox script", `"env"`, `"shell": {"scripts": {"build": "go build ./..."}}, "env"`, `script "build": move it`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := matchingProject(t)
			p.Devbox = []byte(strings.Replace(devbox, tc.from, tc.to, 1))
			assertOneProblem(t, verify(t, p), tc.want)
		})
	}
}

func TestGoModsGoLineMustBeTheMinorVersionDevboxPins(t *testing.T) {
	for _, tc := range []struct {
		name, goLine string
		agrees       bool
	}{
		{"the same minor version", "go 1.27", true},
		{"a patch release of it", "go 1.27.0", true},
		{"an older minor version", "go 1.26", false},
		{"a newer minor version", "go 1.28", false},
		{"a prefix of the minor version", "go 1.2", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := matchingProject(t)
			p.GoMod = []byte(strings.Replace(goMod, "go 1.27", tc.goLine, 1))
			problems := verify(t, p)
			if tc.agrees {
				if problems != nil {
					t.Fatalf("got %q", problems)
				}
				return
			}
			assertOneProblem(t, problems, "go mod edit -go=1.27")
		})
	}
}

func TestAGoModWithoutAGoLineIsReported(t *testing.T) {
	p := matchingProject(t)
	p.GoMod = []byte("module example.com/app\n")
	assertOneProblem(t, verify(t, p), "go.mod has no go line")
}

func TestAMissingDevboxIsReported(t *testing.T) {
	p := matchingProject(t)
	p.Devbox = nil
	assertOneProblem(t, verify(t, p), "devbox.json is missing")
}

func TestDirectoriesTheLayoutGuideRulesOutAreReported(t *testing.T) {
	for _, dir := range []string{"src", "pkg", "internal/util", "internal/common", "internal/app/helpers"} {
		t.Run(dir, func(t *testing.T) {
			p := matchingProject(t)
			p.Dirs = append(p.Dirs, dir)
			assertOneProblem(t, verify(t, p), dir+"/")
		})
	}
}

func TestAPureLibraryCantHaveAdaptersOrCommands(t *testing.T) {
	for _, dir := range []string{"adapters", "cmd"} {
		t.Run(dir, func(t *testing.T) {
			p := matchingProject(t)
			config, err := baseline.GolangciConfig("example.com/app", baseline.PureLibrary)
			if err != nil {
				t.Fatal(err)
			}
			p.Golangci = config
			p.Dirs = []string{dir}
			assertOneProblem(t, verify(t, p), "pure library")
		})
	}
}

func TestAnAdaptersConfigLimitsSizeEverywhereAndHasNoPurityRules(t *testing.T) {
	config, err := baseline.GolangciConfig("example.com/adapters", baseline.Adapters)
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"\n    - depguard\n", "\n    - forbidigo\n", "forbidigo:", "path-except"} {
		if strings.Contains(string(config), unwanted) {
			t.Errorf("config contains %q:\n%s", unwanted, config)
		}
	}
}

func TestAnAdaptersModuleCantHaveCommandsOrACore(t *testing.T) {
	for _, dir := range []string{"cmd", "internal/core"} {
		t.Run(dir, func(t *testing.T) {
			p := matchingProject(t)
			config, err := baseline.GolangciConfig("example.com/app", baseline.Adapters)
			if err != nil {
				t.Fatal(err)
			}
			p.Golangci = config
			p.Dirs = []string{dir}
			assertOneProblem(t, verify(t, p), "adapters module")
		})
	}
}

func TestPkgIsAllowedBelowTheRoot(t *testing.T) {
	p := matchingProject(t)
	p.Dirs = append(p.Dirs, "internal/core/pkg")
	if problems := verify(t, p); problems != nil {
		t.Fatalf("got %q", problems)
	}
}

func assertOneProblem(t *testing.T, problems []string, want string) {
	t.Helper()
	if len(problems) != 1 || !slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, want) }) {
		t.Fatalf("got %q, want one problem containing %q", problems, want)
	}
}

func TestRolesFollowThePath(t *testing.T) {
	for _, tc := range []struct {
		kind baseline.Kind
		dir  string
		want baseline.Role
	}{
		{baseline.Command, "internal/core/greeting", baseline.Core},
		{baseline.Command, "internal/app", baseline.Shell},
		{baseline.Command, "", baseline.Shell},
		{baseline.Command, "internal/adapters/clock", baseline.Adapter},
		{baseline.Command, "adapters/clock", baseline.Adapter},
		{baseline.Command, "cmd/app", baseline.Root},
		{baseline.Command, "internal/corefake", baseline.Shell},
		{baseline.PureLibrary, "", baseline.Core},
		{baseline.PureLibrary, "parse", baseline.Core},
		{baseline.Adapters, "", baseline.Adapter},
		{baseline.Adapters, "clock", baseline.Adapter},
	} {
		t.Run(string(tc.kind)+":"+tc.dir, func(t *testing.T) {
			if got := baseline.RoleOf(tc.kind, tc.dir); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRolesHaveNames(t *testing.T) {
	for role, want := range map[baseline.Role]string{
		baseline.Core: "core", baseline.Shell: "shell", baseline.Adapter: "adapter",
		baseline.Root: "composition root", baseline.Role(0): "Role(0)",
	} {
		if got := role.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

// goList is `go list -deps -json` output for a command whose core reaches a
// third-party package that does I/O through another one, and whose adapter
// uses an I/O package directly.
const goList = `{"ImportPath": "os", "Standard": true}
{"ImportPath": "fmt", "Standard": true, "Imports": ["os"]}
{"ImportPath": "example.com/leaf", "Imports": ["net/http", "net/url"], "Module": {"Path": "example.com/leaf"}}
{"ImportPath": "example.com/mid", "Imports": ["example.com/leaf", "fmt"], "Module": {"Path": "example.com/mid"}}
{"ImportPath": "example.com/pure", "Imports": ["fmt"], "Module": {"Path": "example.com/pure"}}
{"ImportPath": "example.com/sdk", "Imports": ["os"], "Module": {"Path": "example.com/sdk"}}
{"ImportPath": "example.com/app/internal/core/rates", "Imports": ["example.com/mid", "example.com/pure", "fmt"], "Module": {"Path": "example.com/app"}}
{"ImportPath": "example.com/app/internal/app", "Imports": ["example.com/app/internal/core/rates"], "Module": {"Path": "example.com/app"}}
{"ImportPath": "example.com/app/internal/adapters/store", "Imports": ["example.com/sdk", "os"], "Module": {"Path": "example.com/app"}}
`

func parsedGoList(t *testing.T) []baseline.Package {
	t.Helper()
	packages, err := baseline.ParsePackages([]byte(goList))
	if err != nil {
		t.Fatal(err)
	}
	return packages
}

func TestIODependenciesOfTheCoreAreReportedEvenIndirectly(t *testing.T) {
	problems := baseline.DependencyProblems("example.com/app", baseline.Command, parsedGoList(t))
	want := "example.com/app/internal/core/rates depends on example.com/leaf, which imports net/http"
	if len(problems) != 1 || !strings.HasPrefix(problems[0], want) {
		t.Fatalf("got %q, want one problem starting %q", problems, want)
	}
}

func TestVerifyReportsIODependencies(t *testing.T) {
	p := matchingProject(t)
	p.Packages = parsedGoList(t)
	assertOneProblem(t, verify(t, p), "example.com/leaf")
}

func TestUnreadableGoListIsAnError(t *testing.T) {
	if _, err := baseline.ParsePackages([]byte("{")); err == nil {
		t.Fatal("got no error")
	}
}

const goTestOutput = "?   \texample.com/app/internal/core/types\t[no test files]\n" +
	"\texample.com/app/internal/core/untested\t\tcoverage: 0.0% of statements\n" +
	"ok  \texample.com/app/internal/core/rates\t0.004s\tcoverage: 92.5% of statements\n" +
	"ok  \texample.com/app/internal/core/thin\t0.004s\tcoverage: 61.0% of statements\n" +
	"\texample.com/app/internal/adapters/store\t\tcoverage: 0.0% of statements\n"

func TestCoverageIsReadPerPackage(t *testing.T) {
	got := baseline.ParseCoverage([]byte(goTestOutput))
	want := map[string]float64{
		"example.com/app/internal/core/untested":  0,
		"example.com/app/internal/core/rates":     92.5,
		"example.com/app/internal/core/thin":      61,
		"example.com/app/internal/adapters/store": 0,
	}
	if !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestOnlyCorePackagesBelowTheMinimumAreReported(t *testing.T) {
	var packages []baseline.Package
	for _, name := range []string{"core/types", "core/untested", "core/rates", "core/thin", "adapters/store"} {
		p := baseline.Package{ImportPath: "example.com/app/internal/" + name}
		p.Module = &struct{ Path, Dir string }{"example.com/app", ""}
		packages = append(packages, p)
	}
	problems := baseline.CoverageProblems("example.com/app", baseline.Command, packages, baseline.ParseCoverage([]byte(goTestOutput)))
	if len(problems) != 2 || !strings.Contains(problems[0], "core/thin") || !strings.Contains(problems[1], "core/untested") {
		t.Fatalf("got %q, want core/thin and core/untested", problems)
	}
}

func TestImpurePackagesIncludeThoseReachingIOThroughOthers(t *testing.T) {
	got := baseline.ImpurePackages("example.com/app", parsedGoList(t))
	want := map[string]bool{"example.com/leaf": true, "example.com/mid": true, "example.com/sdk": true}
	if !maps.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestModuleDirsListsOtherModules(t *testing.T) {
	packages, err := baseline.ParsePackages([]byte(`{"ImportPath": "example.com/app", "Module": {"Path": "example.com/app", "Dir": "/src/app"}}
{"ImportPath": "example.com/lib/a", "Module": {"Path": "example.com/lib", "Dir": "/cache/lib"}}
{"ImportPath": "example.com/lib/b", "Module": {"Path": "example.com/lib", "Dir": "/cache/lib"}}
{"ImportPath": "fmt", "Standard": true}
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := baseline.ModuleDirs("example.com/app", packages); !maps.Equal(got, map[string]string{"example.com/lib": "/cache/lib"}) {
		t.Fatalf("got %v", got)
	}
}
