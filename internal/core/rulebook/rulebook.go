// Package rulebook describes how go-baseline enforces each clause of the
// code rulebook (github.com/ianflanagan1/code-rulebook), as
// the coverage listing the rulebook checks.
package rulebook

import (
	"encoding/json"
	"fmt"
	"maps"
)

// ListingPath is where the listing is kept, from the module's root, for the
// rulebook to import.
const ListingPath = "rulebook-coverage.json"

// COUPLED:coverage-listing-shape:[same-shape; //github.com/ianflanagan1/code-rulebook/schema/coverage.schema.json; //github.com/ianflanagan1/code-rulebook/internal/core/rulebook/model.go;
// //github.com/ianflanagan1/ts-baseline/src/core/listing.ts;
// //github.com/ianflanagan1/php-baseline/src/Core/RulebookListing.php] fields, check kinds and coverage levels
type listing struct {
	Language string `json:"language"`
	Baseline string `json:"baseline"`
	// Roles maps each of the rulebook's roles to how the checks tell it.
	Roles map[string]string `json:"roles"`
	// ReviewedRoles names the roles whose code's changes need a person's
	// review.
	ReviewedRoles []string         `json:"reviewed-roles"`
	Checks        map[string]check `json:"checks"`
	Clauses       map[string]claim `json:"clauses"`
}

type check struct {
	Kind string `json:"kind"`
	Tool string `json:"tool,omitempty"`
	// OnFinding is "review" for a check of provisional clauses, whose
	// findings go to a person and fail nothing. Empty, a finding fails.
	OnFinding string `json:"on-finding,omitempty"`
	Summary   string `json:"summary"`
}

type claim struct {
	// Case is, for a clause that varies by a trait, Go's value of the trait.
	Case     string   `json:"case,omitempty"`
	Coverage string   `json:"coverage"`
	Checks   []string `json:"checks,omitempty"`
	Gap      string   `json:"gap,omitempty"`
	Where    string   `json:"where,omitempty"`
	Proof    []string `json:"proof,omitempty"`
	// Intent and Reason are set for every coverage but full. Reason says why
	// the check was declined.
	Intent string `json:"intent,omitempty"`
	Reason string `json:"reason,omitempty"`
	Note   string `json:"note,omitempty"`
}

// COUPLED:coverage-listing-shape:END

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
	return []string{test("TestConformance/" + clause)}
}

// byDirectory starts the entry of each role the checks tell by where a
// package is. The rulebook reads a role from a declaration's own code, which
// no check here does yet.
const byDirectory = "By directory, not from the code: "

// helpers is how the checks tell a shell role's helpers from its members.
const helpers = " Its helpers are the unexported identifiers of its packages, and the packages under an internal/ directory inside it, " +
	"which Go lets only packages under that role's directory import; every other package there is a member."

// coreDirectory is how the checks tell the core's two roles, which is the
// same for both.
const coreDirectory = "every package under a core/ directory, internal/core/ in an application and core/ in a library, is checked as the core. " +
	"The checks don't tell a decision from a value: one directory holds both, and no check applies a rule to one and not the other. It has no helpers."

func roles() map[string]string {
	return map[string]string{
		"entrypoint": byDirectory + "every package under cmd/ is checked as entrypoint code, and no check looks for func main. " +
			"Each main package is an entrypoint. Its helpers are the unexported identifiers of the main packages " +
			"and every other package under cmd/, which belongs under cmd/internal/ so that only the entrypoints can import it.",
		"driving-adapter": byDirectory + "every package under an adapters/driving/ directory is checked as driving-adapter code. " +
			"The checks don't tell a driving adapter's entry from the helpers that serve its contract, such as its controllers." + helpers,
		"driven-adapter": byDirectory + "every package under an adapters/driven/ directory is checked as driven-adapter code, " +
			"whether or not it implements a port." + helpers,
		"port": "The checks don't tell a port as a declaration. An interface declared in a package under orchestrators/ is checked as orchestrator code, " +
			"with the rest of its package. Only the portbranch analyzer treats a port apart: in an orchestrator package it takes as read through a port " +
			"the result of a method called on any interface the core doesn't declare, and of a call to any function value, " +
			"whether or not a service holds it as a collaborator. It has no helpers.",
		"orchestrator": byDirectory + "every package under an orchestrators/ directory is checked as orchestrator code, " +
			"the ports it declares included." + helpers,
		"decision": byDirectory + coreDirectory,
		"value":    byDirectory + coreDirectory,
		"fake": "The checks don't tell a fake: it has no directory, and no check looks for a service that implements a port and that only test code builds. " +
			"One declared in a _test.go file is test code to every check. One declared in any other file is checked with the role its package's directory gives, " +
			"and in a module with a program verify's deadcode step reports it, since no program calls it.",
	}
}

func goBaseline() listing {
	return listing{
		Language:      "go",
		Baseline:      "github.com/ianflanagan1/go-baseline",
		Roles:         roles(),
		ReviewedRoles: []string{"entrypoint", "driving-adapter", "driven-adapter"},
		Checks:        checks(),
		Clauses:       claims(),
	}
}

func checks() map[string]check {
	lint := func(summary string) check { return check{Kind: "tool", Tool: "golangci-lint", Summary: summary} }
	custom := func(summary string) check { return check{Kind: "custom", Summary: summary} }
	language := func(summary string) check { return check{Kind: "language", Summary: summary} }
	return map[string]check{
		"role-of":                   custom("RoleOf gives each package one role from its directory alone, in every module. The core is one role to it, for decision and value code alike."),
		"verify-roles":              custom("verify reports a package outside every role's directory, which has no role."),
		"verify-gitignore":          custom("verify reports a .gitignore without the line /cache/, or a project without one."),
		"depguard-core":             lint("depguard's core rule: core packages import only the core, pure standard-library packages and third-party packages."),
		"depguard-orchestrators":    lint("depguard's orchestrators rule: orchestrators import no adapter and no I/O package."),
		"depguard-driving-adapters": lint("depguard's driving-adapters rule: driving adapters import no driven adapter of the project's."),
		"depguard-driven-adapters":  lint("depguard's driven-adapters rule: of the project's packages, driven adapters import only the core and other driven adapters."),
		"entrypoint-unimportable":   language("An entrypoint is a main package, which Go lets no package import."),
		"internal-packages":         language("A package under an internal/ directory can be imported only by packages under that directory's parent."),
		"forbidigo-effects": lint("forbidigo bans the clock, sleeping, timers and tickers (time.Now, Since, Until, Sleep, After, AfterFunc, Tick, NewTimer and NewTicker), " +
			"printing and the global loggers, outside adapters/, cmd/ and test files."),
		"gochecknoglobals":    lint("Bans package-level variables everywhere."),
		"gochecknoinits":      lint("Bans init functions everywhere."),
		"gosec-unsafe":        lint("gosec's G103 reports each use of unsafe.Pointer and the unsafe package's functions, outside test files."),
		"gosec-weak-random":   lint("gosec's G404 reports calls of the generators in math/rand and math/rand/v2, in every role, outside test files."),
		"verify-dependencies": custom("verify follows go list -deps and reports a third-party package of the core or an orchestrator that imports an I/O package at any depth."),
		"coreboundary":        custom("Analyzer: a value passed to the core as an interface, function, type argument or field has a pure concrete type."),
		"portbranch":          custom("Analyzer: follows values read through a port and reports orchestrator conditions that depend on them."),
		"configread":          custom("Analyzer: environment variables, flags and os.Args are read only in cmd/."),
		"funlen-edges":        lint("funlen limits functions in adapters and cmd/ to 50 lines and 30 statements."),
		"gocognit":            lint("Cognitive complexity at most 10 everywhere but cmd/."),
		"import-cycles":       language("The Go compiler rejects import cycles between packages."),
		"depguard-containers": lint("depguard's containers rule: no package imports a dependency-injection container (fx, dig, do, di or golobby/container)."),
		"gochecksumtype":      lint("gochecksumtype: an interface marked //sumtype:decl is sealed, and a type switch on it names every variant, with default-signifies-exhaustive off."),
		"exhaustive":          lint("exhaustive: a switch on an enum names every value, with default-signifies-exhaustive off."),
		"sumtypedecl":         custom("Analyzer: every sealed interface, one with an unexported method, is marked //sumtype:decl."),
		"nodefault":           custom("Analyzer: a switch on the project's own enum, or a type switch on its own sealed interface, has no default arm."),
		"unexported-methods":  language("Only a type in an interface's own package can have its unexported methods, so only that package can implement an interface with one."),
		"errcheck":            lint("errcheck: no returned error is dropped, unless discarded explicitly with _."),
		"ineffassign":         lint("ineffassign: no assigned value goes unread."),
		"unused-variables":    language("The Go compiler rejects a local variable that is declared and never used."),
		"go-syntax":           language("Go has no default or optional parameters and no field initialisers; the parser rejects the syntax other languages use for them."),
		"static-typing": language("Go is statically typed, at one level that no setting weakens: the compiler rejects a parameter, result, field or collection element with no declared type, " +
			"and a value of type any used as another type without a type assertion or a type switch."),
		"no-eval":                  language("Go has no function or construct that runs a string as code: its built-in functions are a fixed list, and none takes code."),
		"modifies-parameter":       lint("revive's modifies-parameter rule, at confidence 0.5: no function assigns to one of its parameters, including from a closure."),
		"revive-error-return":      lint("revive's error-return rule: an error is the last of a function's results."),
		"revive-empty-block":       lint("revive's empty-block rule: no block is empty, the branch of an if among them."),
		"modernize-slicescontains": lint("modernize's slicescontains analyzer: a loop that only searches a slice for an element equal to a value is to be a call of slices.Contains."),
		"golangci-fmt":             lint("golangci-lint fmt --diff, which the lint task runs, fails on any file gofumpt or goimports would change."),
		"verify-config":            custom("verify reports a .golangci.yml or Taskfile.yml that differs from the one the baseline generates."),
	}
}

// heldState is the gap in the effect clauses: Go has no way to stop a
// method writing its receiver's fields.
const heldState = "Keeping state that changes across calls in an instance's fields (held-state) isn't detected."

// unlistedEffects is the gap in the clauses about third-party code: the
// dependency check follows only imports of the I/O packages.
const unlistedEffects = "Only imports of the I/O packages are followed, so third-party code that reads the clock, prints or logs " +
	"through the time, fmt, log or log/slog packages, which aren't on that list, isn't detected."

// mainImportable is the gap in the clauses about referencing entrypoint
// code: only its main packages can't be imported.
const mainImportable = "A package under cmd/ that isn't a main package and isn't under an internal/ directory, such as cmd/wiring, " +
	"can be imported; depguard's rule for this role doesn't name cmd/."

// oneCore is the gap in the clauses that turn on the difference between
// decision code and value code, which one directory holds.
const oneCore = "The checks don't tell decision code from value code: both are under core/."

// secondConfig is the gap in the clauses that rest on verify keeping
// .golangci.yml as the baseline generates it.
const secondConfig = "golangci-lint reads a .golangci.json, .golangci.toml or .golangci.yaml ahead of .golangci.yml, and verify compares only .golangci.yml, " +
	"so a project can leave that file as generated and put other settings in one of those."

// errorsGuide is where BASELINE.md states the error-handling conventions.
const errorsGuide = "BASELINE.md, Errors"

// none is a clause nothing checks and BASELINE.md doesn't state, which a
// check is to enforce.
func none() claim { return claim{Coverage: "none", Intent: "planned"} }

// noneYet is none with a note.
func noneYet(note string) claim { return claim{Coverage: "none", Intent: "planned", Note: note} }

// test names a test in internal/core/baseline.
func test(name string) string { return "internal/core/baseline: " + name }

// analyzerTest names a test in internal/core/analyzers.
func analyzerTest(name string) string { return "internal/core/analyzers: " + name }

// entrypointProof shows the compiler keeping other roles' code from
// importing a main package.
func entrypointProof() string { return test("TestTheCompilerLetsNoPackageImportAnEntrypoint") }

func claims() map[string]claim {
	all := map[string]claim{}
	for _, category := range []map[string]claim{
		roleClaims(), coreClaims(), shellClaims(), portClaims(), operationClaims(),
		familyClaims(), constructionClaims(), typeClaims(), changeClaims(), reachClaims(),
		errorClaims(), persistenceClaims(), projectClaims(), testingClaims(),
	} {
		maps.Copy(all, category)
	}
	return all
}

// roleClaims are the architecture clauses about which role a declaration
// has and which roles' code it may reference.
func roleClaims() map[string]claim {
	return map[string]claim{
		"every-declaration-has-one-role": {
			Coverage: "partial", Checks: []string{"role-of", "verify-roles"},
			Gap: "A role is taken from the directory a package is in, not from what a declaration is, holds, implements and is built by. " +
				"So a declaration in the wrong directory is checked as the wrong role, a service that fits no role, such as one holding both a port and a database client, isn't reported, " +
				"and decision code and value code are one role. Only a package outside every role's directory is reported.",
			Proof:  []string{test("TestRolesFollowThePath"), test("TestAPackageOutsideEveryRolesDirectoryHasNoRole"), test("TestAPackageWithNoRoleIsReported")},
			Intent: "planned",
		},
		"a-unit-of-code-holds-one-role": noneYet("A package's directory gives every declaration in it the one role, so the checks never see two roles in a package: " +
			"a decision beside a value object under core/, or a pure function beside a store under adapters/driven/, is checked as its directory's role and isn't reported."),
		"dependency-declarations-keep-their-role": {
			Coverage: "partial", Checks: []string{"role-of", "coreboundary", "verify-dependencies"},
			Gap: "A first-party dependency's packages get the roles their directories give, not the roles their code gives, and only the analyzers use them, " +
				"so its core/ counts as the core. depguard's rules name only the project's own module: a project's core can import a dependency's orchestrator, " +
				"and its driving adapter a dependency's driven adapter. Only an I/O import, at any depth, stops the core or an orchestrator importing a dependency's adapter.",
			Proof:  []string{test("TestRolesFollowThePath"), analyzerTest("TestCoreBoundary"), test("TestIODependenciesOfTheCoreAreReportedEvenIndirectly")},
			Intent: "planned",
		},
		"other-roles-reach-only-members": {
			Coverage: "partial", Checks: []string{"internal-packages"},
			Gap: "A helper in a package that isn't under its role's internal/ directory counts as a member, so other roles' code can import it. " +
				"So does a package under cmd/ outside cmd/internal/.",
			Proof:  []string{test("TestTheCompilerKeepsARolesHelpersFromOtherRoles")},
			Intent: "declined",
			Reason: "Whether a package is a member, such as one use case, or a helper is a matter of what it's for, which no static check can see; the baseline lets its location decide.",
			Note:   "Go is stricter than the clause: an entrypoint can't import a helper under another role's internal/ directory to build it.",
		},
		"no-reference-cycles": {
			Coverage: "full", Checks: []string{"import-cycles"}, Proof: []string{test("TestTheCompilerRejectsImportCycles")},
			Note: "A sealed union and its variants are one package, which unexported-methods holds them to, so a cycle through a variant is a cycle between packages.",
		},
		"driving-adapter-code-references-only-orchestrators-and-the-core": {
			Coverage: "partial", Checks: []string{"depguard-driving-adapters", "entrypoint-unimportable"},
			Gap: "A port is declared in its orchestrator's package, which a driving adapter may import, so one that references a port isn't reported. " +
				"Nor is a driven adapter of another module, such as go-shared-code: the rule names only the project's own, whether or not the adapter is a system adapter that driving-adapter code may call. " +
				mainImportable,
			Proof:  append(conformance("driving-adapter-code-references-only-orchestrators-and-the-core"), entrypointProof()),
			Intent: "planned",
			Note:   "The fixture internal/adapters/driving/timed shows the system stopwatch, which the clause allows, passing.",
		},
		"driven-adapter-code-references-only-ports-and-values": {
			Coverage: "partial", Checks: []string{"depguard-driven-adapters", "entrypoint-unimportable"},
			Gap:    oneCore + " A driven adapter that references or calls a decision function there isn't reported.",
			Proof:  append(conformance("driven-adapter-code-references-only-ports-and-values"), entrypointProof()),
			Intent: "planned",
			Note:   "Go is stricter than the clause on ports: a driven adapter can't import the orchestrator package that declares the port it implements, and needn't.",
		},
		"driving-adapter-code-calls-no-driven-adapter": {
			Coverage: "partial", Checks: []string{"depguard-driving-adapters"},
			Gap: "A driving adapter that declares an interface and is given a driven adapter through it by the entrypoint isn't caught; only importing one of the project's own is. " +
				"One given the system clock isn't reported either: no check tells the system adapters that driving-adapter code may call, the stopwatch and the deadlines, from the rest.",
			Proof: conformance("driving-adapter-code-calls-no-driven-adapter"), Intent: "planned",
		},
		"driven-adapter-code-calls-no-orchestrator": {
			Coverage: "partial", Checks: []string{"depguard-driven-adapters"},
			Gap:   "A driven adapter that declares an interface and is given an orchestrator through it by the entrypoint isn't caught; only importing one is.",
			Proof: conformance("driven-adapter-code-calls-no-orchestrator"), Intent: "planned",
		},
		"driving-adapter-code-only-reports-what-a-decision-returns": noneYet(
			"Needs an analyzer that follows what decision code returns through driving-adapter code, as portbranch follows a port's value through an orchestrator."),
		"orchestrator-code-references-no-entrypoint-or-adapter-code": {
			Coverage: "partial", Checks: []string{"depguard-orchestrators", "entrypoint-unimportable"}, Gap: mainImportable,
			Proof:  append(conformance("orchestrator-code-references-no-entrypoint-or-adapter-code"), entrypointProof()),
			Intent: "planned",
		},
		"entrypoint-code-calls-only-driving-adapters-and-values": noneYet(
			"Needs a check of what cmd/ calls, other than constructors and Close, and one that tells value code from decision code."),
		"value-code-references-only-value-code": {
			Coverage: "partial", Checks: []string{"depguard-core"},
			Gap:   oneCore + " A core type that references a decision function isn't reported; only a reference to a package in another role's directory is.",
			Proof: conformance("value-code-references-only-value-code"), Intent: "planned",
		},
		"value-code-decides-nothing": noneYet("No check tells value code from decision code, and none looks at what a core type's method works out."),
	}
}

// coreClaims are the architecture clauses about the core's purity and what
// crosses into it.
func coreClaims() map[string]claim {
	dependencies := []string{test("TestIODependenciesOfTheCoreAreReportedEvenIndirectly"), test("TestImpurePackagesIncludeThoseReachingIOThroughOthers")}
	return map[string]claim{
		"core-references-only-core": {Coverage: "full", Checks: []string{"depguard-core"}, Proof: conformance("core-references-only-core")},
		"core-does-nothing-impure": {
			Coverage: "partial", Checks: []string{"depguard-core", "forbidigo-effects", "gochecknoglobals"},
			Gap: heldState + " Nor is the machine's local time zone: the core may call time.Unix, whose result is in the local zone, " +
				"and use time.Local and time.LoadLocation.",
			Proof: conformance("core-does-nothing-impure"), Intent: "planned",
			Note: "depguard bans the I/O packages, which include os (environment, files, standard streams), math/rand and crypto/rand; forbidigo bans the clock and printing, which live in pure packages, in a method as in a function.",
		},
		"core-calls-reach-nothing-impure": {
			Coverage: "partial", Checks: []string{"verify-dependencies"}, Gap: unlistedEffects, Proof: dependencies, Intent: "planned",
			Note: "go list -deps resolves the versions the project builds with, and following every import reaches further than following every call.",
		},
		"core-loading-does-nothing-impure": {
			Coverage: "partial", Checks: []string{"gochecknoinits", "gochecknoglobals", "verify-dependencies"},
			Gap:   "A third-party package's load-time code is followed only through its imports. " + unlistedEffects,
			Proof: append(conformance("core-loading-does-nothing-impure"), dependencies...), Intent: "planned",
			Note: "The core's own load-time code would be its init functions and package-level variables' initialisers, which the linters ban.",
		},
		"core-receives-only-pure-values": {Coverage: "full", Checks: []string{"coreboundary"}, Proof: []string{analyzerTest("TestCoreBoundary")}},
		"the-core-takes-no-plain-data":   none(),
		"only-adapter-and-entrypoint-code-references-plain-data-types": noneYet(
			"Needs a check that tells a plain-data type, a struct with exported fields, and reports a reference to one from the core, an orchestrator or a port."),
		"unknown-types-stay-in-adapter-and-entrypoint-code": noneYet("Needs a check that reports any, and a collection of it, declared in the core, an orchestrator or a port."),
	}
}

// shellClaims are the architecture clauses about orchestrators, adapters and
// entrypoints.
func shellClaims() map[string]claim {
	return map[string]claim{
		"orchestrator-code-performs-no-effects": {
			Coverage: "partial", Checks: []string{"depguard-orchestrators", "forbidigo-effects", "gochecknoglobals"},
			Gap: heldState, Proof: conformance("orchestrator-code-performs-no-effects"), Intent: "planned",
		},
		"orchestrator-code-reaches-nothing-impure": {
			Coverage: "partial", Checks: []string{"verify-dependencies"}, Gap: unlistedEffects,
			Proof: []string{
				test("TestIODependenciesOfAnOrchestratorAreReportedEvenIndirectly"), test("TestVerifyReportsIODependencies"),
				test("TestImpurePackagesIncludeThoseReachingIOThroughOthers"),
			},
			Intent: "planned",
			Note:   "Following every import reaches a package whose load-time code alone runs, as well as every function that could be called.",
		},
		"orchestrator-code-branches-only-on-decisions": {
			Coverage: "partial", Checks: []string{"portbranch"},
			Gap: "Any call into the core clears a port's value, since the checks don't tell a core type's method from a decision function: " +
				"a condition on what a method derives from a port's value, such as whether a list of orders is empty, isn't reported. " +
				"Computing with a port's value without branching on it, such as sorting it, isn't detected either.",
			Proof: []string{analyzerTest("TestPortBranch")}, Intent: "planned",
		},
		"orchestrator-code-runs-no-loop-until-stopped": noneYet("Needs a check that reports a loop in an orchestrator package with no condition, or whose condition is a port's stop signal."),
		"orchestrator-is-given-ports": {
			Coverage: "partial", Checks: []string{"depguard-orchestrators", "depguard-containers", "verify-dependencies"},
			Gap: "A collaborator of a pure standard-library or third-party type that isn't a port, such as a *slog.Logger or an io.Writer, isn't reported, " +
				"nor is an interface the orchestrator didn't declare.",
			Proof: conformance("orchestrator-is-given-ports"), Intent: "planned",
			Note: "An orchestrator can't import an adapter, a container or a client library that does I/O, so it can't take one.",
		},
		"orchestrator-takes-and-returns-signature-types":                  none(),
		"decisions-driving-adapters-call-take-and-return-signature-types": none(),
		"adapter-code-function-size-is-limited": {
			Coverage: "full", Checks: []string{"funlen-edges", "gocognit"}, Proof: conformance("adapter-code-function-size-is-limited"),
		},
		"only-entrypoint-code-reads-configuration": {
			Coverage: "full", Checks: []string{"configread"},
			Proof: []string{analyzerTest("TestConfigRead"), analyzerTest("TestTheEnvAdapterMayReadConfiguration")},
		},
		"entrypoint-code-function-size-is-limited": {
			Coverage: "full", Checks: []string{"funlen-edges"}, Proof: conformance("entrypoint-code-function-size-is-limited"),
		},
		"entrypoint-code-wires-without-a-container": {
			Coverage: "partial", Checks: []string{"depguard-containers"},
			Gap: "Only the common container libraries are banned. Wire, which generates the wiring as code, is allowed, " +
				"and a service locator the project writes itself isn't detected.",
			Proof:  conformance("entrypoint-code-wires-without-a-container"),
			Intent: "declined",
			Reason: "An import check sees only the libraries it lists, and a hand-written locator is a map looked up by name, which no static check tells from other maps.",
		},
	}
}

// portClaims are the architecture clauses about ports.
func portClaims() map[string]claim {
	return map[string]claim{
		"port-methods-take-and-return-signature-types": none(),
		"port-is-declared-by-its-caller": {
			Coverage: "full", Checks: []string{"depguard-orchestrators"}, Proof: conformance("port-is-declared-by-its-caller"),
			Note: "An orchestrator can't import an adapter, so it can't take an interface an adapter declares.",
		},
		"port-units-of-code-follow-interface-implementation": {
			Case: "structural", Coverage: "partial", Checks: []string{"depguard-driven-adapters"},
			Gap:   "A package of interfaces that other orchestrators import is an orchestrator like any other, so a ports package isn't caught.",
			Proof: conformance("port-units-of-code-follow-interface-implementation"), Intent: "planned",
			Note: "A driven adapter can't import the orchestrator that declares the port it implements.",
		},
		"port-is-named-in-domain-terms": {
			Coverage: "convention-only", Where: "BASELINE.md, Databases", Intent: "declined",
			Reason: "Whether a name is in the domain's terms, or a method is shaped for one thing its caller does, is a judgement no static check can make.",
		},
		"port-methods-return-no-interfaces": {Coverage: "convention-only", Where: "BASELINE.md, Databases", Intent: "planned"},
		"port-methods-need-no-call-order":   none(),
		"writes-that-belong-together-are-one-port-call": noneYet(
			"Which writes a use case needs made together isn't in the code, so a check can reach little of this."),
	}
}

// operationClaims are the architecture clauses about the impure operations
// each role's code may perform.
func operationClaims() map[string]claim {
	return map[string]claim{
		"adapters-and-entrypoints-perform-only-their-roles-operations": {
			Coverage: "partial", Checks: []string{"configread", "gochecknoglobals"},
			Gap: "Only configuration read outside cmd/, and package-level state, are reported. depguard and forbidigo leave every I/O package to adapters/ and cmd/ alike, " +
				"so a driving adapter that calls out over the network or opens a database, a driven adapter that serves requests, and an entrypoint that does either, aren't.",
			Proof: append(conformance("adapters-and-entrypoints-perform-only-their-roles-operations"), analyzerTest("TestConfigRead")), Intent: "planned",
		},
		"adapters-and-entrypoints-reach-only-their-roles-operations": noneYet(
			"verify-dependencies follows third-party imports only from the core and orchestrators, and asks only whether they reach I/O, not which operation."),
		"reserved-operations-are-performed-only-by-their-system-adapters": {
			Coverage: "partial", Checks: []string{"forbidigo-effects", "depguard-core", "depguard-orchestrators", "gosec-weak-random"},
			Gap: "In the core and orchestrators the clock, the timers and both random packages are banned. Under adapters/ and cmd/ only math/rand is reported: " +
				"an adapter or entrypoint that reads the clock, measures a duration, starts a timer or ticker or draws from crypto/rand isn't. " +
				"Nowhere is a time zone read (time.Local, time.LoadLocation, the local result of time.Unix), a context given a deadline outside the core, or an ID laid out by hand, reported.",
			Proof: conformance("reserved-operations-are-performed-only-by-their-system-adapters"), Intent: "planned",
			Note: "No check exempts a system adapter by name: they are in go-shared-code, which a project's checks don't lint. " +
				"gosec's message says to use crypto/rand, which is reserved too, and BASELINE.md still has a project write its own clock adapter.",
		},
		"no-third-party-feature-performs-a-reserved-operation":                         noneYet("go-baseline has no dependency list to name each release unit's banned features."),
		"standard-library-and-third-party-code-takes-sources-from-the-system-adapters": none(),
		"nothing-sleeps": {
			Coverage: "partial", Checks: []string{"forbidigo-effects"},
			Gap: "forbidigo is off under adapters/ and cmd/ and in test files, so an adapter or entrypoint that sleeps isn't reported, and neither is a third-party function that sleeps. " +
				"In the core and orchestrators it reports time.Sleep, time.After and the timers, but not a wait on a context whose deadline is all there is to wait for.",
			Proof: conformance("nothing-sleeps"), Intent: "planned",
			Note: "forbidigo's message for time.Sleep says to take the clock through a port, which this clause rules out as well; BASELINE.md's retry loop sleeps through one.",
		},
		"outside-facing-code-names-no-internal-type":                      none(),
		"external-driving-adapters-get-no-internal-values":                none(),
		"external-driving-adapters-get-no-internal-values-from-decisions": none(),
		"external-driven-adapters-take-no-internal-values":                none(),
		"only-decisions-and-internal-adapters-read-internal-values":       none(),
		"the-internal-wrapper-shows-nothing-it-holds": noneYet(
			"It is to be shown by tests of the wrapper in go-shared-code, one for each of its text, debug, JSON and logged forms, not by a check of projects."),
	}
}

// familyClaims are the design clauses about closed families, open
// interfaces and the methods a type has.
func familyClaims() map[string]claim {
	sealing := []string{test("TestAnInterfaceWithAnUnexportedMethodIsImplementedOnlyInItsPackage")}
	return map[string]claim{
		"closed-families-are-sealed": {
			Coverage: "partial", Checks: []string{"gochecksumtype", "sumtypedecl"},
			Gap:    "A family whose interface has only exported methods isn't recognised as one, so nothing stops another package implementing it.",
			Proof:  append(conformance("closed-families-are-sealed"), analyzerTest("TestSumTypeDecl")),
			Intent: "declined",
			Reason: "Whether an interface's variants are all its package's own is a design decision, which no static check can see until the interface is sealed.",
		},
		"sealed-variants-share-their-unit-of-code": {Coverage: "full", Checks: []string{"unexported-methods"}, Proof: sealing},
		"matches-on-closed-families-are-exhaustive": {
			Coverage: "full", Checks: []string{"gochecksumtype", "exhaustive", "sumtypedecl"},
			Proof: append(conformance("matches-on-closed-families-are-exhaustive"), analyzerTest("TestSumTypeDecl")),
			Note:  "sumtypedecl makes every sealed interface carry the marker gochecksumtype looks for.",
		},
		"matches-on-closed-families-have-no-default-arm": {
			Coverage: "full", Checks: []string{"exhaustive", "gochecksumtype", "nodefault"},
			Proof: append(conformance("matches-on-closed-families-have-no-default-arm"), analyzerTest("TestNoDefault")),
			Note: "A default arm doesn't excuse a missing case, and nodefault reports one beside cases that name every value, whatever it holds. A switch on another module's type may have one. " +
				"The clause's exception doesn't arise in Go: the panic that ends a function after an exhaustive switch goes after the switch.",
		},
		"open-interfaces-are-not-matched-by-implementation": noneYet("Needs a check of type switches and assertions on interfaces that aren't sealed, other than error."),
		"no-port-is-sealed": {
			Coverage: "full", Checks: []string{"unexported-methods"}, Proof: sealing,
			Note: "A port sealed with an unexported method can't be implemented by a driven adapter or a fake in another package, so the program doesn't compile.",
		},
		"types-declare-every-method-they-have": noneYet("Needs a check that reports a struct's embedded field; an interface may embed another."),
		"every-type-has-one-shape":             none(),
		"no-hooks-reached-through-catch-all-parameters": noneYet(
			"Needs a check that reports a declared String, GoString, Format, LogValue, MarshalJSON, UnmarshalJSON, MarshalText or UnmarshalText method."),
		"nothing-runs-when-a-value-is-collected": noneYet("runtime.SetFinalizer and runtime.AddCleanup aren't banned."),
	}
}

// constructionClaims are the design clauses about how values are built and
// what functions take.
func constructionClaims() map[string]claim {
	syntax := func(clause string) []string { return []string{test("TestGoHasNoSyntaxFor/" + clause)} }
	return map[string]claim{
		"no-parameter-default-values": {Coverage: "full", Checks: []string{"go-syntax"}, Proof: syntax("no-parameter-default-values")},
		"no-optional-parameters": {
			Coverage: "full", Checks: []string{"go-syntax"}, Proof: syntax("no-optional-parameters"),
			Note: "A variadic parameter can be left out. It's how Go passes a list, and isn't checked.",
		},
		"constructors-only-store-their-arguments": noneYet("Go's constructors are New functions, which nothing tells apart from other functions yet."),
		"field-initialisers-are-constants":        {Coverage: "full", Checks: []string{"go-syntax"}, Proof: syntax("field-initialisers-are-constants")},
		"core-types-have-factory-functions": {
			Case: "hidden-fields", Coverage: "convention-only", Where: "BASELINE.md, Core types", Intent: "planned",
		},
		"core-types-are-built-only-by-their-factory-functions": {
			Coverage: "convention-only", Where: "BASELINE.md, Core types", Intent: "planned",
		},
		"no-standard-library-or-third-party-code-builds-core-types": noneYet("reflect.New, reflect.NewAt and reflect.Zero aren't banned."),
		"one-factory-function-calls-each-constructor":               none(),
		"factory-functions-take-signature-types":                    none(),
		"no-boolean-parameters":                                     none(),
		"no-boolean-fields":                                         none(),
		"core-type-fields-are-not-nullable":                         none(),
		"no-positional-groups": {
			Case: "multiple-results", Coverage: "partial", Checks: []string{"revive-error-return"},
			Gap: "Only an error that isn't the last result is reported. Two values, two values and an error, a value with both a boolean and an error, " +
				"and a core type returned by value beside an error or a boolean from outside its package, aren't.",
			Proof: conformance("no-positional-groups"), Intent: "planned",
		},
	}
}

// typeClaims are the design clauses the type checker bears on.
func typeClaims() map[string]claim {
	typed := func(clause string) claim {
		return claim{Coverage: "full", Checks: []string{"static-typing"}, Proof: []string{test("TestTheCompilerRejects/" + clause)}}
	}
	return map[string]claim{
		"parameters-are-typed":                                typed("parameters-are-typed"),
		"return-types-are-declared-where-called-from-outside": typed("return-types-are-declared-where-called-from-outside"),
		"fields-are-typed":                                    typed("fields-are-typed"),
		"collection-types-name-their-element-type":            typed("collection-types-name-their-element-type"),
		"no-type-turns-checking-off": withNote(typed("no-type-turns-checking-off"),
			"unsafe.Pointer converts to any pointer type untested, which no-cast-asserts-an-unchecked-type covers."),
		"no-cast-asserts-an-unchecked-type": {
			Coverage: "partial", Checks: []string{"gosec-unsafe"},
			Gap:   "A single-result type assertion, which panics on a value of another type, isn't reported. Test files aren't checked.",
			Proof: conformance("no-cast-asserts-an-unchecked-type"), Intent: "planned",
		},
		"values-have-immutable-types": none(),
	}
}

// changeClaims are the design clauses about values that don't change.
func changeClaims() map[string]claim {
	return map[string]claim{
		"arguments-are-not-mutated":                none(),
		"fields-are-assigned-only-at-construction": none(),
		"field-collections-are-not-changed":        none(),
		"parameters-are-not-reassigned": {
			Coverage: "full", Checks: []string{"modifies-parameter"}, Proof: conformance("parameters-are-not-reassigned"),
		},
		"core-locals-are-not-reassigned-once-read": {Case: "unsupported", Coverage: "none", Intent: "planned"},
		"built-values-are-not-changed":             none(),
		"shared-collections-are-not-changed": {
			Case: "unsupported", Coverage: "partial", Checks: []string{"modifies-parameter"},
			Gap: "Only an append assigned back to the slice parameter itself is reported, as a parameter reassigned. An append to a passed slice under another name, " +
				"a write to an element, a sort, a delete or a clear, of a collection the function was passed or has already shared, isn't.",
			Proof: conformance("shared-collections-are-not-changed"), Intent: "planned",
		},
		"references-are-not-compared-by-identity": none(),
		"collections-are-not-keyed-by-instance":   none(),
		"loops-compute-no-value-an-operation-names": {
			Coverage: "partial", Checks: []string{"modernize-slicescontains"},
			Gap: "Only a loop that searches a slice for an element equal to a value is reported. One that fills a list or a map, counts, totals, keeps the least or greatest, " +
				"or tests each element with anything but ==, isn't.",
			Proof: conformance("loops-compute-no-value-an-operation-names"), Intent: "planned",
		},
		"collection-operations-take-pure-functions":         none(),
		"folds-compute-no-value-a-narrower-operation-names": noneYet("Go's shared collection operations, the fold among them, don't exist yet."),
		"collection-operations-come-from-one-place": noneYet(
			"Utility libraries such as samber/lo aren't banned, and nothing reports a project's own generic collection function."),
	}
}

// reachClaims are the design clauses about code reaching only what it
// names.
func reachClaims() map[string]claim {
	return map[string]claim{
		"no-code-is-run-from-text":             {Coverage: "full", Checks: []string{"no-eval"}, Proof: []string{test("TestGoHasNoBuiltinThatRunsText")}},
		"nothing-is-called-by-a-computed-name": noneYet("reflect's Value.MethodByName and Value.Call aren't banned."),
		"nothing-is-imported-by-a-computed-path": {
			Coverage: "partial", Checks: []string{"depguard-core", "depguard-orchestrators"},
			Gap:   "depguard keeps the plugin package from the core and orchestrators only: an adapter or entrypoint may load a plugin.",
			Proof: conformance("nothing-is-imported-by-a-computed-path"), Intent: "planned",
		},
		"visibility-is-not-bypassed": {
			Coverage: "partial", Checks: []string{"gosec-unsafe"},
			Gap:   "A //go:linkname directive isn't reported, nor an unexported field read through reflect alone, which needs no unsafe. Test files aren't checked.",
			Proof: conformance("visibility-is-not-bypassed"), Intent: "planned",
		},
	}
}

func errorClaims() map[string]claim {
	errorsConvention := func(intent, reason, note string) claim {
		return claim{Coverage: "convention-only", Where: errorsGuide, Intent: intent, Reason: reason, Note: note}
	}
	reported := "errcheck reports an error a driving adapter drops, under returned-failures-are-used; what it then tells its caller isn't checked, and BASELINE.md shows it only by example."
	return map[string]claim{
		"core-throws-nothing": withCase(errorsConvention("planned", "",
			"BASELINE.md says the core never panics. The case allows the one panic that ends a function after an exhaustive switch."), "needs-statement"),
		"core-catches-expected-failures-it-calls": errorsConvention("declined",
			"Whether a third-party function panics for an expected failure is in its documentation, not its types, so no static check can tell which calls need a recover.", ""),
		"driving-adapter-catches-each-invocation": errorsConvention("planned", "",
			"No check can find where every framework starts an invocation; one for net/http, reporting an http.Server whose handler isn't wrapped in the project's catch-all, is planned. "+
				"BASELINE.md doesn't yet speak of a function given to the system timer."),
		"driving-adapter-code-catches-everything-only-around-invocations": errorsConvention("planned", "",
			"Needs a verify step reporting recover in adapters/driving/ outside the function the project marks as its catch-all."),
		"invocation-catch-all-reports-a-failure": errorsConvention("declined",
			"What a catch-all writes to its caller and its log isn't decidable statically.", ""),
		"invocation-catch-all-keeps-internals-from-outside-callers": errorsConvention("declined",
			"What a catch-all writes to its caller isn't decidable statically.", ""),
		"driving-adapter-reports-returned-failures":           noneYet(reported),
		"driving-adapter-reports-failures-a-decision-returns": noneYet(reported),
		"entrypoint-catches-what-escapes": errorsConvention("planned", "",
			"Needs a verify step reporting a main that doesn't follow BASELINE.md's pattern: recover, and os.Exit(1) on the error run returns."),
		"error-types-of-other-roles-are-not-matched": {
			Coverage: "partial", Checks: []string{"depguard-core", "depguard-orchestrators", "depguard-driving-adapters", "depguard-driven-adapters"},
			Gap: "Only an error type in a package the matching code can't import is caught. One that an orchestrator's package declares, matched in a driving adapter; " +
				"any error type matched in cmd/; and one a first-party dependency declares in another role, aren't.",
			Proof: conformance("error-types-of-other-roles-are-not-matched"), Intent: "planned",
			Note: "Needs an analyzer reporting errors.Is and errors.As whose target another role's code declares. " +
				"BASELINE.md, Errors, is stricter than the clause: it keeps a driving adapter from matching a database driver's error, which the clause now allows.",
		},
		"orchestrator-code-throws-nothing":  withCase(errorsConvention("planned", "", ""), "needs-statement"),
		"orchestrator-code-catches-nothing": errorsConvention("planned", "", ""),
		"port-methods-return-expected-failures": errorsConvention("declined",
			"A port method can't throw; it returns error, and whether its result also carries every expected failure isn't decidable.", ""),
		"driven-adapter-code-converts-expected-failures": errorsConvention("planned", "",
			"Which errors a dependency returns for expected failures isn't known to any tool, but the half that says driven-adapter code recovers nothing can be checked."),
		"returned-failures-are-used": {
			Coverage: "partial", Checks: []string{"errcheck", "ineffassign", "unused-variables"},
			Gap:   "A call that drops all of a (T, bool) result, as a statement, isn't reported.",
			Proof: conformance("returned-failures-are-used"), Intent: "planned",
		},
		"failure-branches-act": {
			Coverage: "partial", Checks: []string{"revive-empty-block"},
			Gap:   "A failure branch that returns success, such as return nil in a function that returns an error, isn't reported.",
			Proof: conformance("failure-branches-act"), Intent: "planned",
		},
		"explicit-discards-give-a-reason": noneYet("errcheck accepts every discard to the blank identifier, with a comment or without."),
		"sentinels-report-only-absence":   none(),
		"sentinels-are-null-or-a-found-flag": noneYet(
			"BASELINE.md, Core types, states the half about a core type outside its package, which returns *T; nothing states that no value of a result's own type means failure."),
	}
}

func persistenceClaims() map[string]claim {
	claims := map[string]claim{}
	for _, clause := range []string{
		"sql-text-comes-only-from-the-query-compiler", "queries-are-query-data", "queries-are-built-in-driven-adapter-code",
		"each-project-table-has-an-id-type", "id-types-accept-only-uuidv7s", "ids-are-held-as-their-id-types",
		"only-the-shared-id-unit-of-code-uses-third-party-uuid-code",
		"enums-are-stored-as-their-values", "enum-columns-admit-only-their-enums-values",
		"every-project-table-has-a-uuid-id-primary-key", "id-columns-reject-other-uuid-versions", "ids-say-nothing-about-when-rows-were-made",
		"public-id-columns-are-unique-uuidv4s", "each-public-id-has-a-type", "public-id-types-accept-only-uuidv4s",
		"nullable-columns-hold-only-variant-data", "variant-data-columns-are-constrained-to-their-variants",
		"every-stored-value-comes-from-the-query", "queries-perform-no-impure-operation", "the-schema-supplies-no-values",
		"no-value-depends-on-session-settings",
	} {
		claims[clause] = none()
	}
	return claims
}

// projectClaims are the formatting and project clauses.
func projectClaims() map[string]claim {
	return map[string]claim{
		"check-fails-on-unformatted-files": {
			Coverage: "partial", Checks: []string{"golangci-fmt", "verify-config"},
			Gap:    secondConfig + " One whose formatters exclude the project's files makes golangci-lint fmt --diff pass on any of them.",
			Proof:  []string{test("TestTheLintTaskFailsOnUnformattedFiles"), test("TestTaskfileDriftIsReported")},
			Intent: "planned",
		},
		"formatter-settings-come-from-the-baseline": {
			Coverage: "partial", Checks: []string{"verify-config"}, Gap: secondConfig,
			Proof: []string{test("TestAnEditedConfigIsReported")}, Intent: "planned",
			Note: "gofumpt and goimports are configured in .golangci.yml, which verify keeps as the baseline generates it.",
		},
		"caches-live-under-the-cache-directory": {
			Coverage: "convention-only", Where: "BASELINE.md, Working on the project", Intent: "declined",
			Reason: "The tools the baseline runs keep their caches outside the project; a check would need to know the cache files of every tool a project adds.",
		},
		"gitignore-ignores-the-cache-directory": {
			Coverage: "full", Checks: []string{"verify-gitignore"},
			Proof: []string{
				test("TestTheCacheDirectoryMustBeIgnored"), test("TestTheCacheLineMayEndWithSpacesOrACarriageReturn"), test("TestAMissingGitignoreIsReported"),
			},
		},
		"type-checker-runs-at-its-strictest": {
			Coverage: "full", Checks: []string{"static-typing"}, Proof: []string{test("TestTheCompilerRejects/type-checker-runs-at-its-strictest")},
			Note: "Go's type checker is its compiler, which has no strictness to turn down. The linters' settings are baseline-settings-are-not-weakened's.",
		},
		"baseline-settings-are-not-weakened": {
			Coverage: "partial", Checks: []string{"verify-config"}, Gap: secondConfig,
			Proof:  []string{test("TestALinterConfigWeakerThanTheBaselinesIsReported"), test("TestAnEditedConfigIsReported")},
			Intent: "planned",
			Note:   "Within .golangci.yml verify is stricter than the clause: a project can't add a linter or tighten a setting there either; that changes in go-baseline.",
		},
		"checks-are-not-suppressed": {
			Coverage: "partial", Checks: []string{"verify-config"},
			Gap: "A //nolint, //lint:ignore or #nosec comment isn't reported; BASELINE.md, Working on the project, only asks for none. " +
				"Nor is a file whose build constraint no analysed build satisfies. " + secondConfig,
			Proof:  []string{test("TestALinterConfigWeakerThanTheBaselinesIsReported")},
			Intent: "planned",
		},
		"check-task-runs-every-baseline-check": {
			Coverage: "partial", Checks: []string{"verify-config"},
			Gap:    "The check task's own lines can't change, but which configuration its golangci-lint steps read can. " + secondConfig,
			Proof:  []string{test("TestACheckTaskThatRunsLessThanTheBaselinesIsReported"), test("TestTaskfileDriftIsReported")},
			Intent: "planned",
			Note: "A project's own tasks are in Taskfile.project.yml, and Task refuses one that redefines a task of Taskfile.yml. " +
				"verify is itself a step of the check task, and nothing outside the repository runs it.",
		},
		"every-third-party-dependency-is-listed": noneYet(
			"go-baseline has no dependency list. verify-dependencies rejects only a third-party package that reaches an I/O package from the core or an orchestrator."),
		"at-most-one-listed-dependency-per-capability": none(),
	}
}

func testingClaims() map[string]claim {
	return map[string]claim{
		"every-port-has-one-set-of-contract-tests":      none(),
		"fake-and-adapter-tests-run-the-contract-tests": none(),
		"check-task-runs-mutation-testing-on-the-core": noneYet(
			"The test task fails a core package whose statement coverage is below 80%, which shows that a line ran, not that a test fails when it changes."),
		"core-tests-reference-only-the-core": noneYet("depguard's rules leave out test files."),
		"every-port-has-one-fake":            none(),
		"only-driven-adapters-and-fakes-stand-in-for-ports": noneYet(
			"BASELINE.md's example declares a fake in an orchestrator's test file, which this clause rules out."),
		"fake-code-references-only-ports-values-and-fakes": none(),
		"fake-code-performs-only-held-state": {
			Coverage: "partial", Checks: []string{"gochecknoglobals"},
			Gap: "No check tells fake code, so only what is banned in every role is caught: a package-level variable. A fake that writes a file, reads the clock, " +
				"returns random IDs or prints isn't reported, in a test file or under adapters/.",
			Proof: conformance("fake-code-performs-only-held-state"), Intent: "planned",
		},
		"only-test-code-references-fake-code": noneYet(
			"In a module with a program, verify's deadcode step reports a fake declared outside a test file, since no program calls it, so there a fake can only be test code today."),
		"orchestrator-test-case-calls-one-use-case-once": none(),
		"no-mocking-libraries":                           noneYet("gomock, mockery and testify's mock package aren't banned."),
		"no-hand-written-scripted-doubles":               none(),
	}
}

// withNote returns c with note.
func withNote(c claim, note string) claim {
	c.Note = note
	return c
}

// withCase returns c as coverage of the case named value, of a clause that
// varies by a trait.
func withCase(c claim, value string) claim {
	c.Case = value
	return c
}
