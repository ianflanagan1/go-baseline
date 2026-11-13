package rulebook_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ianflanagan1/go-baseline/internal/core/rulebook"
)

const moduleRoot = "../../.."

// rulebookRelease is the release of the code rulebook that the listing is
// checked against. Moving to a new release means assessing what it changed.
const rulebookRelease = "v0.3.0"

func TestTheListingFitsThePinnedRulebook(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/ianflanagan1/code-rulebook/cmd/code-rulebook@"+rulebookRelease, "check-listing", moduleRoot)
	cmd.Env = append(os.Environ(), "GOPRIVATE=github.com/ianflanagan1/code-rulebook")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("%s doesn't fit code-rulebook %s: %v\n%s", rulebook.ListingPath, rulebookRelease, err, out)
	}
}

func TestTheCommittedListingIsCurrent(t *testing.T) {
	committed, err := os.ReadFile(filepath.Join(moduleRoot, rulebook.ListingPath))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(committed, rulebook.Listing()) {
		t.Errorf("%s is out of date; run `devbox run -- task rulebook-coverage`", rulebook.ListingPath)
	}
}

func TestEveryProofNamesATestInTheModule(t *testing.T) {
	var l struct {
		Clauses map[string]struct{ Proof []string }
	}
	if err := json.Unmarshal(rulebook.Listing(), &l); err != nil {
		t.Fatal(err)
	}
	for clause, c := range l.Clauses {
		for _, proof := range c.Proof {
			dir, name, ok := strings.Cut(proof, ": ")
			if !ok {
				t.Errorf("%s: proof %q isn't <directory>: <test>", clause, proof)
				continue
			}
			test, subtest, _ := strings.Cut(name, "/")
			source := testSource(t, filepath.Join(moduleRoot, filepath.FromSlash(dir)))
			if !strings.Contains(source, "func "+test+"(t *testing.T)") {
				t.Errorf("%s: %s has no test %s", clause, dir, test)
			}
			if subtest != "" && !strings.Contains(source, strconv.Quote(subtest)) {
				t.Errorf("%s: %s has no subtest %s", clause, dir, name)
			}
		}
	}
}

// testSource returns the test files in dir, joined.
func testSource(t *testing.T, dir string) string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	var source strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		source.Write(data)
	}
	return source.String()
}
