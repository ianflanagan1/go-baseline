// Package rulebook describes how go-baseline enforces each clause of the
// code rulebook (github.com/ianflanagan1/code-rulebook), as
// the coverage listing the rulebook checks.
package rulebook

import (
	"encoding/json"
	"fmt"
)

// ListingPath is where the listing is kept, from the module's root, for the
// rulebook to import.
const ListingPath = "rulebook-coverage.json"

// COUPLED:coverage-listing-shape. = code-rulebook:schema/coverage.schema.json (fields, check kinds and coverage levels)
type listing struct {
	Language string            `json:"language"`
	Baseline string            `json:"baseline"`
	Roles    map[string]string `json:"roles"`
	Checks   map[string]check  `json:"checks"`
	Clauses  map[string]claim  `json:"clauses"`
}

type check struct {
	Kind    string `json:"kind"`
	Tool    string `json:"tool,omitempty"`
	Summary string `json:"summary"`
}

type claim struct {
	// Case is, for a clause that varies by a trait, Go's value of the trait.
	Case     string   `json:"case,omitempty"`
	Coverage string   `json:"coverage"`
	Checks   []string `json:"checks,omitempty"`
	Gap      string   `json:"gap,omitempty"`
	Where    string   `json:"where,omitempty"`
	Proof    []string `json:"proof,omitempty"`
	Note     string   `json:"note,omitempty"`
}

// Listing returns the coverage listing as JSON.
func Listing() []byte {
	data, err := json.MarshalIndent(goBaseline(), "", "  ")
	if err != nil {
		panic(fmt.Sprintf("the listing can't be encoded: %v", err))
	}
	return append(data, '\n')
}

// conformance names a subtest of TestConformance, which shows the generated
// golangci-lint config catching a clause's violations.
func conformance(clause string) []string {
	return []string{"internal/core/baseline: TestConformance/" + clause}
}

func goBaseline() listing {
	return listing{
		Language: "go",
		Baseline: "github.com/ianflanagan1/go-baseline",
		Roles: map[string]string{
			"core":            "Packages under a core/ directory: internal/core/ in an application, core/ in a library.",
			"orchestrator":    "Every other package, outside cmd/ and adapters/.",
			"driving-adapter": "Packages under an adapters/driving/ directory.",
			"driven-adapter":  "Packages under an adapters/driven/ directory.",
			"entrypoint":      "Packages under cmd/.",
		},
		Checks:  checks(),
		Clauses: claims(),
	}
}

func checks() map[string]check {
	lint := func(summary string) check { return check{Kind: "tool", Tool: "golangci-lint", Summary: summary} }
	custom := func(summary string) check { return check{Kind: "custom", Summary: summary} }
	language := func(summary string) check { return check{Kind: "language", Summary: summary} }
	return map[string]check{
		"role-of":                   custom("RoleOf gives each package one role from its directory alone, in every module."),
		"verify-roles":              custom("verify reports a package under adapters/ that is in neither adapters/driving/ nor adapters/driven/, which has no role."),
		"verify-release-unit":       custom("verify reports, in a module with an entrypoint, each other package outside internal/, and in a library with any role but adapters, each adapter that imports another module."),
		"depguard-core":             lint("depguard's core rule: core packages import only the core, pure standard-library packages and third-party packages."),
		"depguard-orchestrators":    lint("depguard's orchestrators rule: orchestrators import no adapter and no I/O package."),
		"depguard-driving-adapters": lint("depguard's driving-adapters rule: driving adapters import no driven adapter."),
		"depguard-driven-adapters":  lint("depguard's driven-adapters rule: driven adapters import only the core and other driven adapters."),
		"entrypoint-unimportable":   language("An entrypoint is a main package, which Go lets no package import."),
		"forbidigo-effects":         lint("forbidigo bans the clock, printing and the global loggers outside adapters and cmd/."),
		"gochecknoglobals":          lint("Bans package-level variables everywhere."),
		"verify-dependencies":       custom("verify follows go list -deps and reports a third-party package of the core or an orchestrator that imports an I/O package at any depth."),
		"coreboundary":              custom("Analyzer: a value passed to the core as an interface, function, type argument or field has a pure concrete type."),
		"portbranch":                custom("Analyzer: follows values read through a port and reports orchestrator conditions that depend on them."),
		"configread":                custom("Analyzer: environment variables, flags and os.Args are read only in cmd/."),
		"funlen-edges":              lint("funlen limits functions in adapters and cmd/ to 50 lines and 30 statements."),
		"gocognit":                  lint("Cognitive complexity at most 10 everywhere but cmd/."),
		"import-cycles":             language("The Go compiler rejects import cycles between packages."),
		"depguard-containers":       lint("depguard's containers rule: no package imports a dependency-injection container (fx, dig, do, di or golobby/container)."),
		"gochecksumtype":            lint("gochecksumtype: an interface marked //sumtype:decl is sealed, and a type switch on it names every variant, with default-signifies-exhaustive off."),
		"exhaustive":                lint("exhaustive: a switch on an enum names every value, with default-signifies-exhaustive off."),
		"sumtypedecl":               custom("Analyzer: every sealed interface, one with an unexported method, is marked //sumtype:decl."),
		"nodefault":                 custom("Analyzer: a switch on the project's own enum, or a type switch on its own sealed interface, has no default arm."),
		"unexported-methods":        language("Only a type in an interface's own package can have its unexported methods, so only that package can implement an interface with one."),
		"errcheck":                  lint("errcheck: no returned error is dropped, unless discarded explicitly with _."),
		"ineffassign":               lint("ineffassign: no assigned value goes unread."),
		"unused-variables":          language("The Go compiler rejects a local variable that is declared and never used."),
		"go-syntax":                 language("Go has no class inheritance, default or optional parameters, or field initialisers; the parser rejects the syntax other languages use for them."),
		"modifies-parameter":        lint("revive's modifies-parameter rule, at confidence 0.5: no function assigns to one of its parameters, including from a closure."),
		"golangci-fmt":              lint("golangci-lint fmt --diff, which the lint task runs, fails on any file gofumpt or goimports would change."),
		"verify-config":             custom("verify reports a .golangci.yml or Taskfile.yml that differs from the one the baseline generates."),
	}
}

// heldState is the gap in the effect clauses: Go has no way to stop a
// method writing its receiver's fields.
const heldState = "Keeping state that changes across calls in an instance's fields (held-state) isn't detected."

func claims() map[string]claim {
	syntax := func(clause string) []string {
		return []string{"internal/core/baseline: TestGoHasNoSyntaxFor/" + clause}
	}
	sealing := []string{"internal/core/baseline: TestAnInterfaceWithAnUnexportedMethodIsImplementedOnlyInItsPackage"}
	return map[string]claim{
		"every-unit-has-one-role": {
			Coverage: "full", Checks: []string{"role-of", "verify-roles"},
			Proof: []string{
				"internal/core/baseline: TestRolesFollowThePath", "internal/core/baseline: TestAnAdapterInNeitherDrivingNorDrivenHasNoRole",
				"internal/core/baseline: TestAPackageWithNoRoleIsReported",
			},
		},
		"dependency-units-keep-their-role": {
			Coverage: "partial", Checks: []string{"role-of", "verify-dependencies"},
			Gap: "The analyzers give a dependency on go-baseline the roles its directories give, so its core counts as core, but depguard doesn't: " +
				"a project's core can import a dependency's orchestrator or adapter, which only an I/O import, at any depth, stops.",
			Proof: []string{"internal/core/baseline: TestIODependenciesOfTheCoreAreReportedEvenIndirectly"},
		},
		"core-imports-only-core": {Coverage: "full", Checks: []string{"depguard-core"}, Proof: conformance("core-imports-only-core")},
		"core-performs-no-effects": {
			Coverage: "partial", Checks: []string{"depguard-core", "forbidigo-effects", "gochecknoglobals"},
			Gap:   heldState,
			Proof: conformance("core-performs-no-effects"),
			Note:  "depguard bans the I/O packages, which include os (environment, files, standard streams), math/rand and crypto/rand; forbidigo bans the clock and printing, which live in pure packages.",
		},
		"core-libraries-are-pure": {
			Coverage: "full", Checks: []string{"verify-dependencies"},
			Proof: []string{"internal/core/baseline: TestIODependenciesOfTheCoreAreReportedEvenIndirectly", "internal/core/baseline: TestImpurePackagesIncludeThoseReachingIOThroughOthers"},
		},
		"core-receives-only-pure-values": {Coverage: "full", Checks: []string{"coreboundary"}, Proof: []string{"internal/core/analyzers: TestCoreBoundary"}},
		"orchestrator-imports-no-adapter": {
			Coverage: "full", Checks: []string{"depguard-orchestrators", "entrypoint-unimportable"}, Proof: conformance("orchestrator-imports-no-adapter"),
		},
		"orchestrator-performs-no-effects": {
			Coverage: "partial", Checks: []string{"depguard-orchestrators", "forbidigo-effects", "gochecknoglobals"},
			Gap: heldState, Proof: conformance("orchestrator-performs-no-effects"),
		},
		"orchestrator-libraries-are-pure": {Coverage: "full", Checks: []string{"verify-dependencies"}, Proof: []string{"internal/core/baseline: TestVerifyReportsIODependencies"}},
		"orchestrator-branches-only-on-core-results": {
			Coverage: "partial", Checks: []string{"portbranch"},
			Gap:   "Computing with a port's value without branching on it, such as sorting it, isn't detected.",
			Proof: []string{"internal/core/analyzers: TestPortBranch"},
		},
		"driving-adapter-imports-only-core-orchestrators-and-driving-adapters": {
			Coverage: "partial", Checks: []string{"depguard-driving-adapters", "entrypoint-unimportable"},
			Gap:   "A port is declared in the orchestrator's package, which a driving adapter may import, so a driving adapter that takes a port isn't caught.",
			Proof: conformance("driving-adapter-imports-only-core-orchestrators-and-driving-adapters"),
		},
		"driven-adapter-imports-only-core-ports-and-driven-adapters": {
			Coverage: "full", Checks: []string{"depguard-driven-adapters", "entrypoint-unimportable"},
			Proof: conformance("driven-adapter-imports-only-core-ports-and-driven-adapters"),
		},
		"driving-adapter-calls-no-driven-adapter": {
			Coverage: "partial", Checks: []string{"depguard-driving-adapters"},
			Gap:   "A driving adapter that declares an interface and is given a driven adapter through it by the entrypoint isn't caught; only importing one is.",
			Proof: conformance("driving-adapter-calls-no-driven-adapter"),
		},
		"driven-adapter-calls-no-orchestrator": {
			Coverage: "partial", Checks: []string{"depguard-driven-adapters"},
			Gap:   "A driven adapter that declares an interface and is given an orchestrator through it by the entrypoint isn't caught; only importing one is.",
			Proof: conformance("driven-adapter-calls-no-orchestrator"),
		},
		"adapter-size-is-limited": {Coverage: "full", Checks: []string{"funlen-edges", "gocognit"}, Proof: conformance("adapter-size-is-limited")},
		"only-entrypoints-read-configuration": {
			Coverage: "full", Checks: []string{"configread"},
			Proof: []string{"internal/core/analyzers: TestConfigRead", "internal/core/analyzers: TestTheEnvAdapterMayReadConfiguration"},
		},
		"entrypoint-size-is-limited": {Coverage: "full", Checks: []string{"funlen-edges"}, Proof: conformance("entrypoint-size-is-limited")},
		"wired-without-a-container": {
			Coverage: "partial", Checks: []string{"depguard-containers"},
			Gap:   "Only the common container libraries are banned. Wire, which generates the wiring as code, is allowed.",
			Proof: conformance("wired-without-a-container"),
		},
		"entrypoint-calls-only-driving-adapters": {Coverage: "none", Note: "Needs a check of what cmd/ calls, other than constructors and Close."},
		"no-import-cycles":                       {Coverage: "full", Checks: []string{"import-cycles"}, Proof: []string{"internal/core/baseline: TestTheCompilerRejectsImportCycles"}},
		"ports-declared-by-their-user": {
			Coverage: "full", Checks: []string{"depguard-orchestrators"}, Proof: conformance("ports-declared-by-their-user"),
			Note: "An orchestrator can't import an adapter, so it can't take an interface an adapter declares.",
		},
		"port-units-follow-typing": {
			Case: "structural", Coverage: "partial", Checks: []string{"depguard-driven-adapters"},
			Gap:   "A package of interfaces that other orchestrators import is an orchestrator like any other, so a ports package isn't caught.",
			Proof: conformance("port-units-follow-typing"),
			Note:  "A driven adapter can't import the orchestrator that declares the port it implements.",
		},
		"ports-named-in-domain-terms": {Coverage: "convention", Where: "BASELINE.md, Databases"},
		"applications-are-never-dependencies": {
			Coverage: "full", Checks: []string{"verify-release-unit"},
			Proof: []string{"internal/core/baseline: TestAModuleWithAnEntrypointIsAnApplication", "internal/core/baseline: TestAnApplicationsPackagesOutsideInternalAreReported"},
			Note:  "Go lets no other module import an internal/ package or a main package, so an application's packages can't be imported.",
		},
		"library-adapters-use-only-the-standard-library": {
			Coverage: "full", Checks: []string{"role-of", "verify-release-unit"},
			Proof: []string{
				"internal/core/baseline: TestALibrarysAdaptersUseOnlyTheStandardLibraryAndItsOwnPackages",
				"internal/core/baseline: TestALibraryOfOnlyAdaptersMayUseAnyLibrary",
			},
		},
		"closed-families-are-sealed": {
			Coverage: "partial", Checks: []string{"gochecksumtype", "sumtypedecl"},
			Gap:   "A family whose interface has only exported methods isn't recognised as one, so nothing stops another package implementing it.",
			Proof: append(conformance("closed-families-are-sealed"), "internal/core/analyzers: TestSumTypeDecl"),
		},
		"sealed-variants-share-their-unit": {Coverage: "full", Checks: []string{"unexported-methods"}, Proof: sealing},
		"matches-on-closed-families-are-exhaustive": {
			Coverage: "full", Checks: []string{"gochecksumtype", "exhaustive", "sumtypedecl"},
			Proof: append(conformance("matches-on-closed-families-are-exhaustive"), "internal/core/analyzers: TestSumTypeDecl"),
			Note:  "sumtypedecl makes every sealed interface carry the marker gochecksumtype looks for.",
		},
		"matches-on-closed-families-have-no-default-arm": {
			Coverage: "full", Checks: []string{"exhaustive", "gochecksumtype", "nodefault"},
			Proof: append(conformance("matches-on-closed-families-have-no-default-arm"), "internal/core/analyzers: TestNoDefault"),
			Note:  "A default arm doesn't excuse a missing case, and nodefault reports one beside cases that name every value. A switch on another module's type may have one. Go needs no catch-all.",
		},
		"open-interfaces-are-not-matched-by-type": {Coverage: "none", Note: "Needs a check of type switches and assertions on interfaces that aren't sealed."},
		"ports-are-open": {
			Coverage: "full", Checks: []string{"unexported-methods"}, Proof: sealing,
			Note: "A port sealed with an unexported method can't be implemented by a driven adapter or a test fake in another package, so the program doesn't compile.",
		},
		"classes-extend-only-allowed-bases": {
			Coverage: "full", Checks: []string{"go-syntax"}, Proof: syntax("classes-extend-only-allowed-bases"),
			Note: "Struct embedding, which promotes the embedded type's methods, is the nearest thing, and isn't checked.",
		},
		"no-parameter-default-values": {Coverage: "full", Checks: []string{"go-syntax"}, Proof: syntax("no-parameter-default-values")},
		"no-optional-parameters": {
			Coverage: "full", Checks: []string{"go-syntax"}, Proof: syntax("no-optional-parameters"),
			Note: "A variadic parameter can be left out. It's how Go passes a list, and isn't checked.",
		},
		"constructors-only-store-their-arguments": {Coverage: "none", Note: "Go's constructors are New functions, which nothing tells apart from other functions yet."},
		"field-initialisers-are-constants":        {Coverage: "full", Checks: []string{"go-syntax"}, Proof: syntax("field-initialisers-are-constants")},
		"invariant-types-are-closed":              {Coverage: "none", Note: "Not decided for Go: see the TODO in the rulebook's constructors-only-store rule."},
		"closed-types-have-no-way-around-their-factories": {
			Coverage: "none", Note: "Not decided for Go: see the TODO in the rulebook's constructors-only-store rule.",
		},
		"arguments-are-not-mutated":                {Coverage: "none"},
		"fields-are-assigned-only-at-construction": {Coverage: "none"},
		"field-collections-are-not-changed":        {Coverage: "none"},
		"parameters-are-not-reassigned": {
			Coverage: "full", Checks: []string{"modifies-parameter"}, Proof: conformance("parameters-are-not-reassigned"),
		},
		"core-locals-are-not-reassigned-once-read": {Case: "unsupported", Coverage: "none"},
		"returned-failures-are-used": {
			Coverage: "partial", Checks: []string{"errcheck", "ineffassign", "unused-variables"},
			Gap:   "A call that drops all of a (T, bool) result, as a statement, isn't reported.",
			Proof: conformance("returned-failures-are-used"),
		},
		"check-fails-on-unformatted-files": {
			Coverage: "full", Checks: []string{"golangci-fmt", "verify-config"},
			Proof: []string{"internal/core/baseline: TestTheLintTaskFailsOnUnformattedFiles", "internal/core/baseline: TestTaskfileDriftIsReported"},
		},
		"formatter-settings-come-from-the-baseline": {
			Coverage: "full", Checks: []string{"verify-config"}, Proof: []string{"internal/core/baseline: TestAnEditedConfigIsReported"},
			Note: "gofumpt and goimports are configured in .golangci.yml, which verify keeps as the baseline generates it.",
		},
	}
}
