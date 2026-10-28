package baseline_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

const goMod = "module example.com/app\n\ngo 1.27\n"

const devbox = `{
  "packages": ["go@1.27.3", "golangci-lint@` + baseline.GolangciLintVersion + `"],
  "env": {"GOTOOLCHAIN": "local"},
  "shell": {
    "scripts": {
      "test":  "go test ./...",
      "lint":  "` + baseline.LintScript + `",
      "check": "` + baseline.CheckScript + `"
    }
  }
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
	config, err := baseline.GolangciConfig("example.com/app")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "- pkg: example.com/app/internal/adapters\n") {
		t.Fatalf("config doesn't deny the module's adapters:\n%s", config)
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
	p.GoMod = []byte("module example.com/renamed\n")
	assertOneProblem(t, verify(t, p), ".golangci.yml differs")
}

func TestDevboxDriftIsReported(t *testing.T) {
	for _, tc := range []struct{ name, from, to, want string }{
		{"go not pinned", `"go@1.27.3"`, `"go@latest"`, "pin Go"},
		{"go pinned to a minor version", `"go@1.27.3"`, `"go@1.27"`, "pin Go"},
		{"another golangci-lint", "golangci-lint@" + baseline.GolangciLintVersion, "golangci-lint@2.0.0", "golangci-lint@"},
		{"toolchain downloads allowed", `"GOTOOLCHAIN": "local"`, `"GOTOOLCHAIN": "auto"`, "GOTOOLCHAIN"},
		{"check without verify", baseline.CheckScript, "devbox run test", `"check"`},
		{"lint without formatting", baseline.LintScript, "golangci-lint run", `"lint"`},
		{"no test script", `"test":  "go test ./...",`, "", `"test"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := matchingProject(t)
			p.Devbox = []byte(strings.Replace(devbox, tc.from, tc.to, 1))
			assertOneProblem(t, verify(t, p), tc.want)
		})
	}
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
