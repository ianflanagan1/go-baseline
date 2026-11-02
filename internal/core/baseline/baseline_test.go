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
	golangci, err := baseline.GolangciConfig("example.com/app")
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

func TestTheGeneratedConfigBansAdaptersFromOrchestratorsByModulePath(t *testing.T) {
	config, err := baseline.GolangciConfig("example.com/app")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "- pkg: example.com/app/internal/adapters\n") {
		t.Fatalf("config doesn't deny the module's adapters:\n%s", config)
	}
}

func TestTheGeneratedConfigBansAdaptersAtTheModuleRootFromOrchestrators(t *testing.T) {
	config, err := baseline.GolangciConfig("example.com/lib")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "- pkg: example.com/lib/adapters\n") {
		t.Fatalf("config doesn't deny the module's adapters:\n%s", config)
	}
}

func TestTheGeneratedConfigRequiresWrappingOnlyOtherModulesErrors(t *testing.T) {
	config, err := baseline.GolangciConfig("example.com/app")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "ignore-package-globs:\n        - example.com/app/*\n") {
		t.Fatalf("config doesn't exempt the module's own errors from wrapcheck:\n%s", config)
	}
}

func TestTheGeneratedConfigLimitsTheComplexityOfEveryPackageButTheEntrypoint(t *testing.T) {
	config, err := baseline.GolangciConfig("example.com/app")
	if err != nil {
		t.Fatal(err)
	}
	// Besides enabling gocognit and exempting tests.
	if got := strings.Count(string(config), "- gocognit\n") - 2; got != 1 {
		t.Fatalf("got %d gocognit exemptions besides tests, want 1:\n%s", got, config)
	}
	if !strings.Contains(string(config), "      - path: (^|/)cmd/\n        linters:\n          - gocognit\n") {
		t.Fatalf("config doesn't exempt only the entrypoint:\n%s", config)
	}
}

func TestTheGeneratedConfigIsRecognisedAsTheBaselines(t *testing.T) {
	config, err := baseline.GolangciConfig("example.com/app")
	if err != nil {
		t.Fatal(err)
	}
	if !baseline.IsGenerated(config) {
		t.Fatalf("config isn't recognised:\n%s", config)
	}
	if baseline.IsGenerated([]byte("version: \"2\"\n")) {
		t.Fatal("a config of the project's own is recognised")
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
	p.Dirs = []string{"adapters", "adapters/driven", "adapters/driven/clock", "core", "core/rates"}
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

// own returns a package of example.com/app at dir, importing imports.
func own(dir string, imports ...string) baseline.Package {
	return baseline.Package{
		ImportPath: strings.TrimSuffix("example.com/app/"+dir, "/"),
		Imports:    imports,
		Module:     &struct{ Path, Dir string }{Path: "example.com/app"},
	}
}

func TestAModuleWithAnEntrypointIsAnApplication(t *testing.T) {
	if !baseline.IsApplication("example.com/app", []baseline.Package{own("cmd/app"), own("internal/core/rates")}) {
		t.Fatal("a module with cmd/app isn't an application")
	}
	if baseline.IsApplication("example.com/app", []baseline.Package{own("core/rates"), own("adapters/driven/clock")}) {
		t.Fatal("a module without an entrypoint is an application")
	}
}

func TestAnApplicationsPackagesOutsideInternalAreReported(t *testing.T) {
	for _, dir := range []string{"", "core/rates", "adapters/driven/clock", "app"} {
		t.Run(dir, func(t *testing.T) {
			problems := baseline.ReleaseUnitProblems("example.com/app", []baseline.Package{
				own("cmd/app"), own("internal/core/rates"), own(dir),
			})
			assertOneProblem(t, problems, "move the package under internal/")
		})
	}
}

func TestALibraryMayKeepAnyRoleOutsideInternal(t *testing.T) {
	problems := baseline.ReleaseUnitProblems("example.com/app", []baseline.Package{
		own("core/rates"), own("rates"), own("adapters/driven/clock", "time"),
	})
	if problems != nil {
		t.Fatalf("got %q", problems)
	}
}

func TestALibrarysAdaptersUseOnlyTheStandardLibraryAndItsOwnPackages(t *testing.T) {
	problems := baseline.ReleaseUnitProblems("example.com/app", []baseline.Package{
		own("core/rates"),
		own("adapters/driven/store", "database/sql", "example.com/app/core/rates", "example.com/pgx"),
	})
	assertOneProblem(t, problems, "example.com/app/adapters/driven/store imports example.com/pgx")
}

func TestALibraryOfOnlyAdaptersMayUseAnyLibrary(t *testing.T) {
	problems := baseline.ReleaseUnitProblems("example.com/app", []baseline.Package{
		own("adapters/driven/store", "example.com/pgx"), own("adapters/driving/server", "example.com/router"),
	})
	if problems != nil {
		t.Fatalf("got %q", problems)
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
		dir  string
		want baseline.Role
	}{
		{"internal/core/greeting", baseline.Core},
		{"core/parse", baseline.Core},
		{"internal/orchestrators/app", baseline.Orchestrator},
		{"orchestrators/fetch", baseline.Orchestrator},
		{"internal/adapters/driving/httpapi", baseline.DrivingAdapter},
		{"internal/adapters/driven/clock", baseline.DrivenAdapter},
		{"adapters/driven/clock", baseline.DrivenAdapter},
		{"cmd/app", baseline.Entrypoint},
	} {
		t.Run(tc.dir, func(t *testing.T) {
			if got, ok := baseline.RoleOf(tc.dir); !ok || got != tc.want {
				t.Fatalf("got %v, %t, want %v", got, ok, tc.want)
			}
		})
	}
}

func TestAPackageOutsideEveryRolesDirectoryHasNoRole(t *testing.T) {
	for _, dir := range []string{
		"", "parse", "internal/app", "internal/corefake", "internal/adapters", "internal/adapters/clock", "internal/adapters/clock/driven",
	} {
		t.Run(dir, func(t *testing.T) {
			if got, ok := baseline.RoleOf(dir); ok {
				t.Fatalf("got %v, want no role", got)
			}
		})
	}
}

func TestAPackageWithNoRoleIsReported(t *testing.T) {
	problems := baseline.RolelessProblems("example.com/app", []baseline.Package{
		own("internal/adapters/clock"), own("internal/adapters/driven/store"),
	})
	assertOneProblem(t, problems, "example.com/app/internal/adapters/clock has no role; move it under core/ if it decides")
}

func TestRolesHaveNames(t *testing.T) {
	for role, want := range map[baseline.Role]string{
		baseline.Core: "core", baseline.Orchestrator: "orchestrator", baseline.DrivingAdapter: "driving adapter",
		baseline.DrivenAdapter: "driven adapter", baseline.Entrypoint: "entrypoint", baseline.Role(0): "Role(0)",
	} {
		if got := role.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

// goList is `go list -deps -json` output for an application whose core reaches a
// third-party package that does I/O through another one, and whose adapter
// uses an I/O package directly.
const goList = `{"ImportPath": "os", "Standard": true}
{"ImportPath": "fmt", "Standard": true, "Imports": ["os"]}
{"ImportPath": "example.com/leaf", "Imports": ["net/http", "net/url"], "Module": {"Path": "example.com/leaf"}}
{"ImportPath": "example.com/mid", "Imports": ["example.com/leaf", "fmt"], "Module": {"Path": "example.com/mid"}}
{"ImportPath": "example.com/pure", "Imports": ["fmt"], "Module": {"Path": "example.com/pure"}}
{"ImportPath": "example.com/sdk", "Imports": ["os"], "Module": {"Path": "example.com/sdk"}}
{"ImportPath": "example.com/app/internal/core/rates", "Imports": ["example.com/mid", "example.com/pure", "fmt"], "Module": {"Path": "example.com/app"}}
{"ImportPath": "example.com/app/internal/orchestrators/app", "Imports": ["example.com/app/internal/core/rates"], "Module": {"Path": "example.com/app"}}
{"ImportPath": "example.com/app/internal/adapters/driven/store", "Imports": ["example.com/sdk", "os"], "Module": {"Path": "example.com/app"}}
{"ImportPath": "example.com/app/cmd/app", "Imports": ["example.com/app/internal/orchestrators/app", "example.com/app/internal/adapters/driven/store"], "Module": {"Path": "example.com/app"}}
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
	problems := baseline.DependencyProblems("example.com/app", parsedGoList(t))
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
	"\texample.com/app/internal/adapters/driven/store\t\tcoverage: 0.0% of statements\n"

func TestCoverageIsReadPerPackage(t *testing.T) {
	got := baseline.ParseCoverage([]byte(goTestOutput))
	want := map[string]float64{
		"example.com/app/internal/core/untested":         0,
		"example.com/app/internal/core/rates":            92.5,
		"example.com/app/internal/core/thin":             61,
		"example.com/app/internal/adapters/driven/store": 0,
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
	problems := baseline.CoverageProblems("example.com/app", packages, baseline.ParseCoverage([]byte(goTestOutput)))
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

func TestEachUnreachableFunctionIsAProblem(t *testing.T) {
	output := "internal/app/app.go:12:6: unreachable func: unused\n" +
		"internal/core/rates/rates.go:30:6: unreachable func: Spare\n"
	problems := baseline.DeadcodeProblems([]byte(output))
	if len(problems) != 2 ||
		!strings.HasPrefix(problems[0], "internal/app/app.go:12:6: unreachable func: unused; ") ||
		!strings.HasPrefix(problems[1], "internal/core/rates/rates.go:30:6: unreachable func: Spare; ") {
		t.Fatalf("got %q", problems)
	}
}

func TestNoUnreachableFunctionsIsNoProblems(t *testing.T) {
	if problems := baseline.DeadcodeProblems(nil); problems != nil {
		t.Fatalf("got %q", problems)
	}
}
