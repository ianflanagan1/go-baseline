// Command go-baseline keeps a Go project on the settings every project
// shares. `write` generates .golangci.yml from go.mod; `verify` fails when
// the project has drifted from the baseline.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ianflanagan1/go-baseline/internal/core/baseline"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "go-baseline:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 1 {
		switch args[0] {
		case "write":
			return write()
		case "verify":
			return verify()
		}
	}
	return errors.New("usage: go-baseline write|verify")
}

func write() error {
	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}
	module, err := baseline.ModulePath(goMod)
	if err != nil {
		return err
	}
	config, err := baseline.GolangciConfig(module)
	if err != nil {
		return err
	}
	if err := os.WriteFile(".golangci.yml", config, 0o644); err != nil {
		return fmt.Errorf("writing .golangci.yml: %w", err)
	}
	return nil
}

func verify() error {
	goMod, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}
	golangci, err := readOptional(".golangci.yml")
	if err != nil {
		return err
	}
	devbox, err := readOptional("devbox.json")
	if err != nil {
		return err
	}
	dirs, err := projectDirs()
	if err != nil {
		return err
	}
	problems, err := baseline.Verify(baseline.Project{GoMod: goMod, Golangci: golangci, Devbox: devbox, Dirs: dirs})
	if err != nil {
		return err
	}
	for _, problem := range problems {
		fmt.Fprintln(os.Stderr, problem)
	}
	if len(problems) > 0 {
		return errors.New("the project differs from the baseline")
	}
	return nil
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
