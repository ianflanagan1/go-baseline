package rulebook_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ianflanagan1/go-baseline/internal/core/rulebook"
)

// checksToBuildPath is the list of analyzers and verify steps go-baseline
// still needs, from the module's root. These tests keep it in step with the
// listing and with the checks that exist.
const checksToBuildPath = "docs/analyzers-and-verify-steps-to-build.md"

// checkDirs are the packages that declare go-baseline's own checks: an
// analyzer is a function in the first, a verify step a function in the second.
func checkDirs() []string {
	return []string{"internal/core/analyzers", "internal/core/baseline"}
}

// ownFields are what an entry says of its check itself, when no other
// document specifies it.
func ownFields() []string {
	return []string{"Reports", "Allows", "Fixtures", "Why no existing tool"}
}

// checkToBuild is one entry of the list: the name its check will be declared
// with, and the text after "- **<Field>:**" on each of its lines.
type checkToBuild struct {
	name   string
	fields map[string]string
}

func TestTheListOfChecksToBuildHasEntries(t *testing.T) {
	if len(checksToBuild(t)) == 0 {
		t.Errorf("%s lists no check; give each a \"### <Name>\" heading", checksToBuildPath)
	}
}

func TestEveryCheckToBuildNamesClausesTheListingDoesNotCoverInFull(t *testing.T) {
	var l struct {
		Clauses map[string]struct{ Coverage string }
	}
	if err := json.Unmarshal(rulebook.Listing(), &l); err != nil {
		t.Fatal(err)
	}
	quoted := regexp.MustCompile("`([a-z0-9-]+)`")
	for _, entry := range checksToBuild(t) {
		clauses := quoted.FindAllStringSubmatch(entry.fields["Clauses"], -1)
		if len(clauses) == 0 {
			t.Errorf("%s names no clause; give it a Clauses line with each clause in backticks", entry.name)
		}
		for _, match := range clauses {
			claim, listed := l.Clauses[match[1]]
			switch {
			case !listed:
				t.Errorf("%s names %s, which the listing doesn't have", entry.name, match[1])
			case claim.Coverage == "full":
				t.Errorf("%s names %s, which the listing covers in full; drop the clause from the entry", entry.name, match[1])
			}
		}
	}
}

func TestEveryCheckToBuildSaysWhatItMustDoOrWhereThatIsSpecified(t *testing.T) {
	for _, entry := range checksToBuild(t) {
		if _, specified := entry.fields["Specified in"]; specified {
			continue
		}
		for _, field := range ownFields() {
			if _, ok := entry.fields[field]; !ok {
				t.Errorf("%s has no %s line, and no Specified in line in its place", entry.name, field)
			}
		}
	}
}

func TestNoCheckToBuildNamesACheckThatExists(t *testing.T) {
	declared := declaredFunctions(t)
	for _, entry := range checksToBuild(t) {
		if dir, exists := declared[strings.ToLower(entry.name)]; exists {
			t.Errorf("%s declares %s; delete its entry from %s", dir, entry.name, checksToBuildPath)
		}
	}
}

// checksToBuild reads the list's entries, each under a "### <Name>" heading.
func checksToBuild(t *testing.T) []checkToBuild {
	t.Helper()
	list, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(checksToBuildPath)))
	if err != nil {
		t.Fatalf("%s can't be read: %v", checksToBuildPath, err)
	}
	heading := regexp.MustCompile(`^### (\w+)$`)
	field := regexp.MustCompile(`^- \*\*([^*]+):\*\* (.+)$`)
	var entries []checkToBuild
	inEntry := false
	seen := map[string]bool{}
	for line := range strings.SplitSeq(string(list), "\n") {
		if match := heading.FindStringSubmatch(line); match != nil {
			if seen[match[1]] {
				t.Errorf("%s has two entries; merge them", match[1])
			}
			seen[match[1]] = true
			entries = append(entries, checkToBuild{name: match[1], fields: map[string]string{}})
			inEntry = true
			continue
		}
		if strings.HasPrefix(line, "#") {
			inEntry = false
			continue
		}
		if match := field.FindStringSubmatch(line); match != nil && inEntry {
			entries[len(entries)-1].fields[match[1]] = match[2]
		}
	}
	return entries
}

// declaredFunctions returns the directory of each function the check
// packages declare, keyed by its name in lower case, so that an entry is
// found whether its check was exported or not.
func declaredFunctions(t *testing.T) map[string]string {
	t.Helper()
	declared := map[string]string{}
	for _, dir := range checkDirs() {
		files, err := filepath.Glob(filepath.Join(moduleRoot, filepath.FromSlash(dir), "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		if len(files) == 0 {
			t.Fatalf("%s has no Go files; name the packages that declare the checks in checkDirs", dir)
		}
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			for _, decl := range parsed.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
					declared[strings.ToLower(fn.Name.Name)] = dir
				}
			}
		}
	}
	return declared
}
