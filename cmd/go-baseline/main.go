// Command go-baseline keeps a Go project on the settings every project
// shares. `write` generates .golangci.yml from go.mod, and BASELINE.md;
// `verify` fails when the project has drifted from the baseline, breaks an
// architecture rule, has an untidy go.mod or calls a known vulnerability;
// `test` runs the tests and fails when a core package's
// coverage is too low; `map` describes the project's architecture.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ianflanagan1/go-baseline/internal/adapters/gopackages"
	"github.com/ianflanagan1/go-baseline/internal/core/analyzers"
	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
	"github.com/ianflanagan1/go-baseline/internal/core/projectmap"
	"golang.org/x/vuln/scan"
)

const usage = "usage: go-baseline write [-kind command|pure-library|adapters] | verify | test | map"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "go-baseline:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "write":
			return write(args[1:])
		case "verify":
			if len(args) == 1 {
				return verify()
			}
		case "test":
			if len(args) == 1 {
				return test()
			}
		case "map":
			if len(args) == 1 {
				return describe()
			}
		}
	}
	return errors.New(usage)
}

// write generates .golangci.yml, BASELINE.md, the CI workflow and the Taskfile. Without -kind, it keeps the
// kind the existing .golangci.yml records.
func write(args []string) error {
	flags := flag.NewFlagSet("write", flag.ContinueOnError)
	kindFlag := flags.String("kind", "", "command, pure-library or adapters")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parsing the write flags: %w", err)
	}
	module, err := readModule()
	if err != nil {
		return err
	}
	kind, err := chooseKind(*kindFlag)
	if err != nil {
		return err
	}
	config, err := baseline.GolangciConfig(module, kind)
	if err != nil {
		return err
	}
	if err := os.WriteFile(".golangci.yml", config, 0o600); err != nil {
		return fmt.Errorf("writing .golangci.yml: %w", err)
	}
	if err := os.WriteFile("BASELINE.md", baseline.Guide(), 0o600); err != nil {
		return fmt.Errorf("writing BASELINE.md: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(baseline.WorkflowPath), 0o750); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(baseline.WorkflowPath), err)
	}
	if err := os.WriteFile(baseline.WorkflowPath, baseline.Workflow(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", baseline.WorkflowPath, err)
	}
	if err := os.WriteFile(baseline.TaskfilePath, baseline.Taskfile(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", baseline.TaskfilePath, err)
	}
	return nil
}

func chooseKind(flagValue string) (baseline.Kind, error) {
	if flagValue != "" {
		return baseline.ParseKind(flagValue)
	}
	golangci, err := readOptional(".golangci.yml")
	if err != nil {
		return "", err
	}
	return baseline.KindOf(golangci)
}

func verify() error {
	p, err := readProject()
	if err != nil {
		return err
	}
	problems, err := baseline.Verify(p)
	if err != nil {
		return err
	}
	diagnostics, err := analyse(p)
	if err != nil {
		return err
	}
	untidy, err := tidyProblems()
	if err != nil {
		return err
	}
	dead, err := deadcode(p.Golangci)
	if err != nil {
		return err
	}
	problems = slices.Concat(problems, diagnostics, untidy, dead)
	if err := report(problems, "the project differs from the baseline"); err != nil {
		return err
	}
	return vulncheck()
}

// readProject reads what Verify checks from the project in the working
// directory.
func readProject() (baseline.Project, error) {
	var p baseline.Project
	var err error
	if p.GoMod, err = os.ReadFile("go.mod"); err != nil {
		return p, fmt.Errorf("reading go.mod: %w", err)
	}
	if p.Golangci, err = readOptional(".golangci.yml"); err != nil {
		return p, err
	}
	if p.Guide, err = readOptional("BASELINE.md"); err != nil {
		return p, err
	}
	if p.Workflow, err = readOptional(baseline.WorkflowPath); err != nil {
		return p, err
	}
	if p.Taskfile, err = readOptional(baseline.TaskfilePath); err != nil {
		return p, err
	}
	if p.Devbox, err = readOptional("devbox.json"); err != nil {
		return p, err
	}
	if p.Dirs, err = projectDirs(); err != nil {
		return p, err
	}
	if p.Packages, err = goList("-deps"); err != nil {
		return p, err
	}
	return p, nil
}

// tidyProblems reports go.mod or go.sum differing from what `go mod tidy`
// would write.
func tidyProblems() ([]string, error) {
	var stdout, stderr strings.Builder
	cmd := exec.Command("go", "mod", "tidy", "-diff")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if stdout.Len() > 0 {
		return []string{"go.mod or go.sum isn't tidy; run `go mod tidy`:\n" + stdout.String()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("go mod tidy: %w: %s", err, stderr.String())
	}
	return nil, nil
}

// deadcode reports the functions that no program in a command module
// reaches. Other kinds are imported by other modules, so whatever they export
// is in use. It runs the deadcode in the x/tools that go-baseline requires.
func deadcode(golangci []byte) ([]string, error) {
	if kind, err := baseline.KindOf(golangci); err != nil || kind != baseline.Command {
		return nil, nil
	}
	var stdout, stderr strings.Builder
	cmd := exec.Command("go", "run", "golang.org/x/tools/cmd/deadcode", "./...")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("deadcode: %w: %s", err, stderr.String())
	}
	return baseline.DeadcodeProblems([]byte(stdout.String())), nil
}

// vulncheck runs govulncheck, which fails when the code calls a function
// with a known vulnerability.
func vulncheck() error {
	cmd := scan.Command(context.Background(), "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting govulncheck: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("govulncheck: %w", err)
	}
	return nil
}

// analyse runs the architecture analyzers. When the kind is missing, Verify
// has already reported it, and there's nothing to analyse against.
func analyse(p baseline.Project) ([]string, error) {
	module, err := baseline.ModulePath(p.GoMod)
	if err != nil {
		return nil, err
	}
	kind, kindErr := baseline.KindOf(p.Golangci)
	if kindErr != nil {
		// Verify has reported it.
		return nil, nil
	}
	pure, err := pureModules(baseline.ModuleDirs(module, p.Packages))
	if err != nil {
		return nil, err
	}
	project := analyzers.Project{Module: module, Kind: kind, Impure: baseline.ImpurePackages(module, p.Packages), PureModules: pure}
	return gopackages.Analyze(".", analyzers.All(project))
}

// pureModules returns the modules, from their directories, whose
// .golangci.yml records the pure-library module kind.
func pureModules(dirs map[string]string) (map[string]bool, error) {
	pure := map[string]bool{}
	for module, dir := range dirs {
		golangci, err := readOptional(filepath.Join(dir, ".golangci.yml"))
		if err != nil {
			return nil, err
		}
		if kind, err := baseline.KindOf(golangci); err == nil && kind == baseline.PureLibrary {
			pure[module] = true
		}
	}
	return pure, nil
}

// describe prints each package's role, and the ports the shell declares
// with the adapters that implement them.
func describe() error {
	module, err := readModule()
	if err != nil {
		return err
	}
	golangci, err := readOptional(".golangci.yml")
	if err != nil {
		return err
	}
	kind, err := baseline.KindOf(golangci)
	if err != nil {
		return err
	}
	pkgs, err := gopackages.Types(".")
	if err != nil {
		return err
	}
	fmt.Print(projectmap.Format(kind, projectmap.Describe(module, kind, pkgs)))
	return nil
}

// test runs the tests with coverage and the race detector, then checks the core's coverage.
func test() error {
	module, err := readModule()
	if err != nil {
		return err
	}
	golangci, err := readOptional(".golangci.yml")
	if err != nil {
		return err
	}
	kind, err := baseline.KindOf(golangci)
	if err != nil {
		return err
	}
	var output bytes.Buffer
	cmd := exec.Command("go", "test", "-race", "-cover", "./...")
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}
	packages, err := goList()
	if err != nil {
		return err
	}
	return report(baseline.CoverageProblems(module, kind, packages, baseline.ParseCoverage(output.Bytes())),
		"the core's tests don't cover enough of it")
}

func report(problems []string, summary string) error {
	for _, problem := range problems {
		fmt.Fprintln(os.Stderr, problem)
	}
	if len(problems) > 0 {
		return errors.New(summary)
	}
	return nil
}

func readModule() (string, error) {
	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}
	return baseline.ModulePath(goMod)
}

// goList describes the module's packages, and with -deps everything they
// depend on, not counting tests.
func goList(flags ...string) ([]baseline.Package, error) {
	args := append([]string{"list", "-json=ImportPath,Standard,Imports,Module"}, flags...)
	var stderr strings.Builder
	cmd := exec.Command("go", append(args, "./...")...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, stderr.String())
	}
	return baseline.ParsePackages(out)
}

// readOptional returns nil, and no error, when the file doesn't exist.
func readOptional(name string) ([]byte, error) {
	content, err := os.ReadFile(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	return content, nil
}

// projectDirs lists the directories under the working directory, skipping
// hidden ones and those Go itself ignores.
func projectDirs() ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || path == "." {
			return nil
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || name == "vendor" {
			return filepath.SkipDir
		}
		dirs = append(dirs, filepath.ToSlash(path))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing directories: %w", err)
	}
	return dirs, nil
}
