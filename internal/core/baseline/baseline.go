// Package baseline holds the settings every Go project shares, and checks a
// project against them.
package baseline

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// GolangciLintVersion is the golangci-lint every project pins in devbox.json,
// the version the generated config is written for.
const GolangciLintVersion = "2.15.0"

// LintScript is the devbox "lint" script every project has.
const LintScript = "golangci-lint run && golangci-lint fmt --diff"

// CheckScript is the devbox "check" script every project has.
const CheckScript = "go tool go-baseline verify && devbox run lint && devbox run test"

// ErrNoModule is returned when go.mod has no module line.
var ErrNoModule = errors.New("go.mod has no module line")

//go:embed golangci.yml.tmpl
var golangciTemplate string

// Project is what Verify reads from a project. A missing file is nil.
type Project struct {
	GoMod    []byte
	Golangci []byte
	Devbox   []byte
	// Dirs lists the project's directories, slash-separated and relative to
	// its root.
	Dirs []string
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

// GolangciConfig returns the .golangci.yml for the module at modulePath.
func GolangciConfig(modulePath string) ([]byte, error) {
	tmpl, err := template.New("golangci").Parse(golangciTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing the golangci-lint template: %w", err)
	}
	var out bytes.Buffer
	err = tmpl.Execute(&out, struct {
		Module   string
		IODenies []string
	}{
		Module: modulePath,
		IODenies: []string{
			"crypto/rand", "database/sql", "io/ioutil", "math/rand", "net", "os", "plugin", "syscall",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("rendering the golangci-lint template: %w", err)
	}
	return out.Bytes(), nil
}

// Verify returns each way the project differs from the baseline, or nil when
// it matches.
func Verify(p Project) ([]string, error) {
	module, err := ModulePath(p.GoMod)
	if err != nil {
		return nil, err
	}
	want, err := GolangciConfig(module)
	if err != nil {
		return nil, err
	}
	var problems []string
	switch {
	case p.Golangci == nil:
		problems = append(problems, ".golangci.yml is missing; run `go tool go-baseline write`")
	case !bytes.Equal(p.Golangci, want):
		problems = append(problems, ".golangci.yml differs from the baseline; run `go tool go-baseline write`")
	}
	problems = append(problems, devboxProblems(p.Devbox)...)
	problems = append(problems, layoutProblems(p.Dirs)...)
	return problems, nil
}

var exactGo = regexp.MustCompile(`^go@\d+\.\d+\.\d+$`)

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

	var problems []string
	var hasGo, hasLint bool
	for _, pkg := range config.Packages {
		hasGo = hasGo || exactGo.MatchString(pkg)
		hasLint = hasLint || pkg == "golangci-lint@"+GolangciLintVersion
	}
	if !hasGo {
		problems = append(problems, "devbox.json must pin Go to an exact version, such as go@1.27.3")
	}
	if !hasLint {
		problems = append(problems, "devbox.json must pin golangci-lint@"+GolangciLintVersion)
	}
	if config.Env["GOTOOLCHAIN"] != "local" {
		problems = append(problems, `devbox.json must set env GOTOOLCHAIN to "local"`)
	}
	for _, script := range []struct{ name, want string }{
		{"lint", LintScript},
		{"check", CheckScript},
	} {
		var got string
		if err := json.Unmarshal(config.Shell.Scripts[script.name], &got); err != nil || got != script.want {
			problems = append(problems, fmt.Sprintf("devbox.json script %q must be %q", script.name, script.want))
		}
	}
	if _, ok := config.Shell.Scripts["test"]; !ok {
		problems = append(problems, `devbox.json must have a "test" script`)
	}
	return problems
}

func layoutProblems(dirs []string) []string {
	var problems []string
	for _, dir := range dirs {
		switch base := path.Base(dir); {
		case dir == "src" || dir == "pkg":
			problems = append(problems, fmt.Sprintf("%s/: no src/ or pkg/ directories; see https://go.dev/doc/modules/layout", dir))
		case base == "util" || base == "common" || base == "helpers":
			problems = append(problems, fmt.Sprintf("%s/: name the package for what it provides, not %q", dir, base))
		}
	}
	return problems
}
