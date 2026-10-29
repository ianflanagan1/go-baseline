// Package baseline holds the settings every Go project shares, and checks a
// project against them.
package baseline

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

// GolangciLintVersion is the golangci-lint every project pins in devbox.json,
// the version the generated config is written for.
const GolangciLintVersion = "2.15.0"

// MinCoreCoverage is the statement coverage, in percent, that each core
// package's tests must reach.
const MinCoreCoverage = 80.0

// ErrNoModule is returned when go.mod has no module line.
var ErrNoModule = errors.New("go.mod has no module line")

// ErrNoKind is returned when .golangci.yml doesn't record the project's kind.
var ErrNoKind = errors.New(".golangci.yml doesn't record the project's kind; run `go tool go-baseline write -kind command|pure-library|adapters`")

//go:embed golangci.yml.tmpl
var golangciTemplate string

//go:embed BASELINE.md
var guide []byte

//go:embed check.yml
var workflow []byte

//go:embed Taskfile.yml
var taskfile []byte

// TaskfilePath is where every project has its Taskfile.
const TaskfilePath = "Taskfile.yml"

// WorkflowPath is where every project has its CI workflow.
const WorkflowPath = ".github/workflows/check.yml"

var (
	exactGo      = regexp.MustCompile(`^go@\d+\.\d+\.\d+$`)
	exactTask    = regexp.MustCompile(`^go-task@\d+\.\d+\.\d+$`)
	kindLine     = regexp.MustCompile(`(?m)^# Module kind: ([a-z-]+)\.`)
	coverageLine = regexp.MustCompile(`^(?:ok|FAIL)?\s+(\S+)\s.*coverage: ([0-9.]+)% of statements`)
)

// Kind is what a module is for: its module kind. It decides each package's role.
type Kind string

const (
	// Command builds programs from cmd/.
	Command Kind = "command"
	// PureLibrary is imported by other modules and does no I/O: every
	// package is core.
	PureLibrary Kind = "pure-library"
	// Adapters is imported by other modules' composition roots and adapters:
	// every package is an adapter.
	Adapters Kind = "adapters"
)

// ParseKind returns the Kind named s.
func ParseKind(s string) (Kind, error) {
	switch kind := Kind(s); kind {
	case Command, PureLibrary, Adapters:
		return kind, nil
	}
	return "", fmt.Errorf("unknown kind %q; want command, pure-library or adapters", s)
}

// KindOf returns the module kind a generated .golangci.yml records.
func KindOf(golangci []byte) (Kind, error) {
	match := kindLine.FindSubmatch(golangci)
	if match == nil {
		return "", ErrNoKind
	}
	return ParseKind(string(match[1]))
}

// Role is the part a package plays in the architecture. Its path decides it.
type Role int

const (
	// Core is the functional core: pure functions over values.
	Core Role = iota + 1
	// Shell gathers inputs through ports, calls the core and acts on the
	// result.
	Shell
	// Adapter implements a port by doing I/O.
	Adapter
	// Root is a command's composition root, which builds adapters and
	// passes them in.
	Root
)

func (r Role) String() string {
	switch r {
	case Core:
		return "core"
	case Shell:
		return "shell"
	case Adapter:
		return "adapter"
	case Root:
		return "composition root"
	}
	return "Role(" + strconv.Itoa(int(r)) + ")"
}

// RoleOf returns the role of the package in dir, which is slash-separated and
// relative to the module root ("" for the root). It matches the paths the
// generated .golangci.yml uses.
func RoleOf(kind Kind, dir string) Role {
	switch kind {
	case PureLibrary:
		return Core
	case Adapters:
		return Adapter
	case Command:
	}
	segments := strings.Split(dir, "/")
	switch {
	case slices.Contains(segments, "cmd"):
		return Root
	case slices.Contains(segments, "adapters"):
		return Adapter
	case strings.Contains("/"+dir+"/", "/internal/core/"):
		return Core
	default:
		return Shell
	}
}

// Package is a package as `go list -json` describes it.
type Package struct {
	ImportPath string
	Standard   bool
	Imports    []string
	Module     *struct{ Path, Dir string }
}

// ModuleDirs returns the directory of each module other than module that
// packages, from `go list -deps -json`, come from.
func ModuleDirs(module string, packages []Package) map[string]string {
	dirs := map[string]string{}
	for _, p := range packages {
		if !p.Standard && p.Module != nil && p.Module.Path != module && p.Module.Dir != "" {
			dirs[p.Module.Path] = p.Module.Dir
		}
	}
	return dirs
}

// ParsePackages reads the output of `go list -json`.
func ParsePackages(goList []byte) ([]Package, error) {
	var packages []Package
	decoder := json.NewDecoder(bytes.NewReader(goList))
	for {
		var p Package
		err := decoder.Decode(&p)
		if errors.Is(err, io.EOF) {
			return packages, nil
		}
		if err != nil {
			return nil, fmt.Errorf("reading go list output: %w", err)
		}
		packages = append(packages, p)
	}
}

// Project is what Verify reads from a project. A missing file is nil.
type Project struct {
	GoMod    []byte
	Golangci []byte
	// Guide is the project's BASELINE.md.
	Guide []byte
	// Workflow is the project's CI workflow, at WorkflowPath.
	Workflow []byte
	// Taskfile is the project's Taskfile, at TaskfilePath.
	Taskfile []byte
	Devbox   []byte
	// Dirs lists the project's directories, slash-separated and relative to
	// its root.
	Dirs []string
	// Packages lists the project's packages and everything they depend on,
	// from `go list -deps -json ./...`.
	Packages []Package
}

// ModulePath returns the module path that go.mod declares.
func ModulePath(goMod []byte) (string, error) {
	for line := range strings.Lines(string(goMod)) {
		fields := strings.Fields(strings.SplitN(line, "//", 2)[0])
		if len(fields) != 2 || fields[0] != "module" {
			continue
		}
		if !strings.HasPrefix(fields[1], `"`) {
			return fields[1], nil
		}
		unquoted, err := strconv.Unquote(fields[1])
		if err != nil {
			return "", fmt.Errorf("go.mod module line: %w", err)
		}
		return unquoted, nil
	}
	return "", ErrNoModule
}

// ioPackages are the standard library packages that do I/O or are
// non-deterministic, matched with their subpackages.
func ioPackages() []string {
	return []string{"crypto/rand", "database/sql", "io/ioutil", "math/rand", "net", "os", "plugin", "syscall"}
}

// ioExceptions are pure packages under ioPackages.
func ioExceptions() []string {
	return []string{"net/mail", "net/netip", "net/url"}
}

// IsIO reports whether the standard library package at importPath does I/O
// or is non-deterministic.
func IsIO(importPath string) bool {
	if slices.Contains(ioExceptions(), importPath) {
		return false
	}
	return slices.ContainsFunc(ioPackages(), func(p string) bool {
		return importPath == p || strings.HasPrefix(importPath, p+"/")
	})
}

// dependencyAllowlist lists third-party packages that import an I/O package
// but are safe for the core and shell, each with the reason. Add one only
// after reading the code that projects call.
func dependencyAllowlist() map[string]string {
	return map[string]string{
		"golang.org/x/tools/go/ssa":              "writes to os.Stdout and os.Stderr only in the debug and sanity-check modes a caller enables",
		"golang.org/x/tools/internal/typeparams": "writes to os.Stderr only for tracing that a false constant disables; its other use of os is in a file excluded from builds",
	}
}

// ImpurePackages returns the third-party packages that import an I/O
// package, directly or through other third-party packages, leaving out the
// allowlist. packages come from `go list -deps -json`.
func ImpurePackages(module string, packages []Package) map[string]bool {
	c := impurity{
		module:  module,
		byPath:  packagesByPath(packages),
		allowed: dependencyAllowlist(),
		impure:  map[string]bool{},
	}
	for _, p := range packages {
		if isThirdParty(p, module) {
			c.isImpure(p.ImportPath)
		}
	}
	maps.DeleteFunc(c.impure, func(_ string, isImpure bool) bool { return !isImpure })
	return c.impure
}

// impurity works out which third-party packages import I/O, remembering
// each answer in impure.
type impurity struct {
	module  string
	byPath  map[string]Package
	allowed map[string]string
	impure  map[string]bool
}

func (c impurity) isImpure(importPath string) bool {
	if result, done := c.impure[importPath]; done {
		return result
	}
	_, isAllowed := c.allowed[importPath]
	result := !isAllowed && slices.ContainsFunc(c.byPath[importPath].Imports, c.importsIO)
	c.impure[importPath] = result
	return result
}

// importsIO reports whether imported is an I/O package or, for a
// third-party package, imports one.
func (c impurity) importsIO(imported string) bool {
	if q, ok := c.byPath[imported]; ok && isThirdParty(q, c.module) {
		return c.isImpure(imported)
	}
	return IsIO(imported)
}

// GolangciConfig returns the .golangci.yml for a project of kind, with the
// module at modulePath.
func GolangciConfig(modulePath string, kind Kind) ([]byte, error) {
	tmpl, err := template.New("golangci").Parse(golangciTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing the golangci-lint template: %w", err)
	}
	var out bytes.Buffer
	err = tmpl.Execute(&out, struct {
		Module       string
		Kind         Kind
		IOPackages   []string
		IOExceptions []string
	}{modulePath, kind, ioPackages(), ioExceptions()})
	if err != nil {
		return nil, fmt.Errorf("rendering the golangci-lint template: %w", err)
	}
	return out.Bytes(), nil
}

// Guide returns the BASELINE.md every project has: the rules the linters
// enforce, written for whoever changes the code, people or AI agents.
func Guide() []byte {
	return slices.Clone(guide)
}

// Workflow returns the GitHub Actions workflow every project has, which runs
// `devbox run -- task check` on each push and pull request.
func Workflow() []byte {
	return slices.Clone(workflow)
}

// Taskfile returns the Taskfile every project has, with the check, lint,
// lint-file, test and integration tasks. It includes the project's own tasks from
// Taskfile.project.yml.
func Taskfile() []byte {
	return slices.Clone(taskfile)
}

// Verify returns each way the project differs from the baseline, or nil when
// it matches.
func Verify(p Project) ([]string, error) {
	module, err := ModulePath(p.GoMod)
	if err != nil {
		return nil, err
	}
	if p.Golangci == nil {
		return []string{".golangci.yml is missing; run `go tool go-baseline write -kind command|pure-library|adapters`"}, nil
	}
	kind, err := KindOf(p.Golangci)
	if err != nil {
		// A missing or unknown kind is a problem to report, not a failure.
		return []string{err.Error()}, nil
	}
	want, err := GolangciConfig(module, kind)
	if err != nil {
		return nil, err
	}
	var problems []string
	if !bytes.Equal(p.Golangci, want) {
		problems = append(problems, ".golangci.yml differs from the baseline; run `go tool go-baseline write`")
	}
	switch {
	case p.Guide == nil:
		problems = append(problems, "BASELINE.md is missing; run `go tool go-baseline write`")
	case !bytes.Equal(p.Guide, guide):
		problems = append(problems, "BASELINE.md differs from the baseline; run `go tool go-baseline write`")
	}
	switch {
	case p.Workflow == nil:
		problems = append(problems, WorkflowPath+" is missing; run `go tool go-baseline write`")
	case !bytes.Equal(p.Workflow, workflow):
		problems = append(problems, WorkflowPath+" differs from the baseline; run `go tool go-baseline write`")
	}
	switch {
	case p.Taskfile == nil:
		problems = append(problems, TaskfilePath+" is missing; run `go tool go-baseline write`")
	case !bytes.Equal(p.Taskfile, taskfile):
		problems = append(problems, TaskfilePath+" differs from the baseline; run `go tool go-baseline write`, and put the project's own tasks in Taskfile.project.yml")
	}
	problems = append(problems, devboxProblems(p.Devbox)...)
	problems = append(problems, layoutProblems(kind, p.Dirs)...)
	problems = append(problems, DependencyProblems(module, kind, p.Packages)...)
	return problems, nil
}

func devboxProblems(devbox []byte) []string {
	if devbox == nil {
		return []string{"devbox.json is missing"}
	}
	var config struct {
		Packages []string          `json:"packages"`
		Env      map[string]string `json:"env"`
		Shell    struct {
			Scripts map[string]json.RawMessage `json:"scripts"`
		} `json:"shell"`
	}
	if err := json.Unmarshal(devbox, &config); err != nil {
		return []string{fmt.Sprintf("devbox.json can't be read as a list of packages, string env and scripts: %v", err)}
	}

	problems := devboxPackageProblems(config.Packages)
	if config.Env["GOTOOLCHAIN"] != "local" {
		problems = append(problems, `devbox.json must set env GOTOOLCHAIN to "local"`)
	}
	for _, name := range slices.Sorted(maps.Keys(config.Shell.Scripts)) {
		problems = append(problems, fmt.Sprintf("devbox.json script %q: move it into a task in Taskfile.project.yml; devbox only pins tools", name))
	}
	return problems
}

func devboxPackageProblems(packages []string) []string {
	var problems []string
	if !slices.ContainsFunc(packages, exactGo.MatchString) {
		problems = append(problems, "devbox.json must pin Go to an exact version, such as go@1.27.3")
	}
	if !slices.Contains(packages, "golangci-lint@"+GolangciLintVersion) {
		problems = append(problems, "devbox.json must pin golangci-lint@"+GolangciLintVersion)
	}
	if !slices.ContainsFunc(packages, exactTask.MatchString) {
		problems = append(problems, "devbox.json must pin go-task, which runs the Taskfile, to an exact version, such as go-task@3.53.1")
	}
	return problems
}

func layoutProblems(kind Kind, dirs []string) []string {
	var problems []string
	for _, dir := range dirs {
		switch base := path.Base(dir); {
		case dir == "src" || dir == "pkg":
			problems = append(problems, fmt.Sprintf("%s/: no src/ or pkg/ directories; see https://go.dev/doc/modules/layout", dir))
		case base == "util" || base == "common" || base == "helpers":
			problems = append(problems, fmt.Sprintf("%s/: name the package for what it provides, not %q", dir, base))
		case kind == PureLibrary && (base == "cmd" || base == "adapters"):
			problems = append(problems, fmt.Sprintf("%s/: a pure library has no %s; put I/O in its caller, or in an adapters module", dir, base))
		case kind == Adapters && (base == "cmd" || base == "core"):
			problems = append(problems, fmt.Sprintf("%s/: an adapters module holds only adapters; put %s in another module", dir, base))
		}
	}
	return problems
}

// DependencyProblems reports each third-party package that a core or shell
// package depends on, directly or through other packages, and that imports
// an I/O package. The standard library is trusted: depguard checks the
// module's own imports of it, and its pure packages import os themselves.
func DependencyProblems(module string, kind Kind, packages []Package) []string {
	r := reach{module: module, byPath: packagesByPath(packages), from: map[string]string{}}
	for _, p := range sortedOwnPackages(module, packages) {
		if role := RoleOf(kind, relativeDir(module, p.ImportPath)); role != Core && role != Shell {
			continue
		}
		for _, imported := range p.Imports {
			r.walk(imported, p.ImportPath)
		}
	}
	return r.ioProblems()
}

// reach finds the third-party packages the module's packages depend on.
// from maps each to the first of the module's packages found to depend on
// it.
type reach struct {
	module string
	byPath map[string]Package
	from   map[string]string
}

func (r reach) walk(importPath, from string) {
	p, ok := r.byPath[importPath]
	if !ok || !isThirdParty(p, r.module) {
		return
	}
	if _, seen := r.from[importPath]; seen {
		return
	}
	r.from[importPath] = from
	for _, imported := range p.Imports {
		r.walk(imported, from)
	}
}

// ioProblems reports each I/O import of a package reached, leaving out the
// allowlist.
func (r reach) ioProblems() []string {
	var problems []string
	allowed := dependencyAllowlist()
	for _, importPath := range slices.Sorted(maps.Keys(r.from)) {
		if _, ok := allowed[importPath]; ok {
			continue
		}
		for _, imported := range r.byPath[importPath].Imports {
			if IsIO(imported) {
				problems = append(problems, fmt.Sprintf(
					"%s depends on %s, which imports %s; the core and shell can't depend on I/O, even indirectly",
					r.from[importPath], importPath, imported))
			}
		}
	}
	return problems
}

// ParseCoverage reads `go test -cover` output and returns each package's
// statement coverage, in percent. Packages without statements are left out.
func ParseCoverage(output []byte) map[string]float64 {
	coverage := map[string]float64{}
	for line := range strings.Lines(string(output)) {
		match := coverageLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if percent, err := strconv.ParseFloat(match[2], 64); err == nil {
			coverage[match[1]] = percent
		}
	}
	return coverage
}

// CoverageProblems reports each core package whose coverage is below
// MinCoreCoverage. packages are the module's own, from `go list -json ./...`.
func CoverageProblems(module string, kind Kind, packages []Package, coverage map[string]float64) []string {
	var problems []string
	for _, p := range sortedOwnPackages(module, packages) {
		if RoleOf(kind, relativeDir(module, p.ImportPath)) != Core {
			continue
		}
		if percent, ok := coverage[p.ImportPath]; ok && percent < MinCoreCoverage {
			problems = append(problems, fmt.Sprintf(
				"%s: core coverage is %.1f%%; it must be at least %.0f%%", p.ImportPath, percent, MinCoreCoverage))
		}
	}
	return problems
}

func packagesByPath(packages []Package) map[string]Package {
	byPath := make(map[string]Package, len(packages))
	for _, p := range packages {
		byPath[p.ImportPath] = p
	}
	return byPath
}

func isThirdParty(p Package, module string) bool {
	return !p.Standard && !isOwn(p, module)
}

func isOwn(p Package, module string) bool {
	return !p.Standard && p.Module != nil && p.Module.Path == module
}

func sortedOwnPackages(module string, packages []Package) []Package {
	var own []Package
	for _, p := range packages {
		if isOwn(p, module) {
			own = append(own, p)
		}
	}
	slices.SortFunc(own, func(a, b Package) int { return strings.Compare(a.ImportPath, b.ImportPath) })
	return own
}

func relativeDir(module, importPath string) string {
	if importPath == module {
		return ""
	}
	return strings.TrimPrefix(importPath, module+"/")
}
