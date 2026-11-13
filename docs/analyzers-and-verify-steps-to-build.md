# Analyzers and verify steps to build

Every analyzer and `verify` step of its own that go-baseline still needs, so that each can be built and tested alone. The tests in `internal/core/rulebook/checkstobuild_test.go` keep the list in step with the listing and with the checks that exist.

## Keeping the list

- **A new check or requirement goes here first.** When a clause needs an analyzer or a `verify` step the baseline doesn't have, or what a planned one must do changes, add or change its entry in the same change. Don't build the check in that change.
- **One check to a session, test-first.** For an analyzer, turn the entry's fixtures into packages under `internal/core/analyzers/testdata/src`, mark each line it must report with a `// want` comment, write its test with `analysistest.Run`, see it fail, then write the analyzer in `internal/core/analyzers` and add it to `All`. For a `verify` step, write its test in `internal/core/baseline`, see it fail, then write the function there and have `verify` report what it returns.
- **Then record it.** Describe the rule in `internal/core/baseline/BASELINE.md`, update the clause's claim in `internal/core/rulebook/rulebook.go`, run `task rulebook-coverage`, and delete the entry here. The tests fail while an entry names a check that exists, or a clause the listing covers in full.
- **Only analyzers and `verify` steps.** A linter or a setting of golangci-lint, a new tool and a check of a real database aren't listed here.

## An entry

The heading is the name the check will be declared with: an analyzer's function in `internal/core/analyzers`, or, where the name ends `Problems`, a `verify` step's function in `internal/core/baseline`. Each entry has these lines:

- **Clauses:** the rulebook clauses it enforces, each in backticks.
- **Reports:** what it must report.
- **Allows:** what it must not report.
- **Fixtures:** the code to prove it with: the clauses' violations, and the code it must pass.
- **Why no existing tool:** why the compiler, golangci-lint's linters and go-baseline's own checks don't reach it.
- **Needs first:** what must exist or be decided before it can be built, if anything.
- **Specified in:** another document that already says what the check must do. An entry with this line may leave out the four lines it replaces.

A check that applies to some roles takes each package's role from `RoleOf`, by its directory, until the role classifier gives it roles. Until then, in these entries, a port is an interface that a package under `orchestrators/` declares, and a core type is a struct, an enum or a sealed interface that a package under `core/` declares.

## Ready to build

### NoPanic

- **Clauses:** `core-throws-nothing`, `orchestrator-code-throws-nothing`
- **Reports:** every call of the built-in `panic` in a package under `core/` or `orchestrators/`, except the one that ends an exhaustive switch: a function's last statement, directly after a `switch` on one of the project's enums or a type switch on one of its sealed interfaces.
- **Allows:** that one `panic`; a `panic` in adapter and entrypoint code. A panic the runtime raises, such as a nil dereference, isn't a call and can't be reported.
- **Fixtures:** both clauses' violations in their `needs-statement` case, a function that ends with a `panic` after a type switch over a sealed interface, and a driven adapter that panics.
- **Why no existing tool:** forbidigo's `^panic$` reports every `panic`, the one after an exhaustive switch too, and with no `//nolint` nothing could allow it.
- **Specified in:** code-rulebook `TO-IMPLEMENT.md`, "Errors: catching and throwing by role".

### MainShape

- **Clauses:** `entrypoint-catches-what-escapes`
- **Reports:** a `func main` of a `main` package under `cmd/` that doesn't follow BASELINE.md's pattern: a deferred function that recovers, prints to standard error and calls `os.Exit` with a non-zero status, then a call of a function that returns an `error`, with a print to standard error and `os.Exit` with a non-zero status in the branch taken when the error isn't nil.
- **Allows:** any name and arguments for the function `main` calls, and any text printed.
- **Fixtures:** the clause's violations, as a `main` that logs `run`'s error and exits with status 0, one that recovers and carries on, and one with no deferred recover; and BASELINE.md's example.
- **Why no existing tool:** no linter reads the shape of `main`.
- **Specified in:** code-rulebook `TO-IMPLEMENT.md`, "Errors: catching and throwing by role".

### ExplicitDiscard

- **Clauses:** `explicit-discards-give-a-reason`
- **Reports:** an assignment that gives a call's `error` result to the blank identifier, such as `_ = f.Close()` or `n, _ := w.Write(p)`, with no comment on its line or the line above.
- **Allows:** a discard with a comment; a blank identifier for a result that isn't an `error`.
- **Fixtures:** the clause's first violation, a commented discard, and a blank identifier for the value of a map lookup.
- **Why no existing tool:** errcheck accepts every discard to the blank identifier, and its `check-blank` setting reports every one, commented or not.
- **Specified in:** code-rulebook `TO-IMPLEMENT.md`, "Errors: failures handled, discarded and reported".

### DroppedResult

- **Clauses:** `returned-failures-are-used`
- **Reports:** a call, as a statement of its own or after `go` or `defer`, of a function whose results are a value and a `bool`.
- **Allows:** a call whose results are assigned, to the blank identifier included; a function with any other results, an `error` among them, which is errcheck's.
- **Fixtures:** a function that returns `(Order, bool)` called as a statement, and the same call with both results assigned.
- **Why no existing tool:** errcheck reports only a dropped `error`, and govet's `unusedresult` only the functions it is given by name.

### ResultList

- **Clauses:** `no-positional-groups`, `sentinels-are-null-or-a-found-flag`
- **Reports:** a function, method, function type or interface method that declares more than two results, or two where the second is neither `error` nor `bool`; and, outside a core type's package, one that returns the core type by value beside an `error` or a `bool`, where it should return a pointer to it.
- **Allows:** one result; a value and an `error`; a value and a `bool`; a method that must match an interface or a function type declared outside the project. An `error` that isn't last is revive's `error-return`.
- **Fixtures:** the `multiple-results` case's violations, the second clause's `(Order, bool)` violation, a driven adapter's method that returns `(*Order, error)`, and a core type's own factory function that returns `(Order, error)`. A value of the result's own type set aside to mean failure, such as `-1` or `""`, can't be told from a value, and stays the gap.
- **Why no existing tool:** revive's `function-result-limit` only counts results, and no linter knows which types are core types.

### BooleanDeclaration

- **Clauses:** `no-boolean-parameters`, `no-boolean-fields`
- **Reports:** a parameter of a function, method, function literal or interface method, and a field of a struct, whose type is `bool`, `*bool`, or a slice, array, map or channel of one.
- **Allows:** a `bool` result; a field of a plain-data type, which is a struct in adapter or entrypoint code whose fields are all exported; a function that must match an interface or a function type declared outside the project.
- **Fixtures:** both clauses' violations that Go can write, a predicate that returns `bool`, and a driving adapter's request struct with an exported boolean field.
- **Why no existing tool:** revive's `flag-parameter` reports only a boolean parameter that an `if` tests, and nothing reports a field by its type.

### NullableCoreField

- **Clauses:** `core-type-fields-are-not-nullable`
- **Reports:** in a package under `core/`, a struct field whose type is a pointer, or a slice, array or map whose elements are pointers; and a composite literal of a struct the package declares that leaves a field of an interface type unset, or sets it to `nil`.
- **Allows:** a pointer parameter or result, such as `*Order` for an order that may be missing; fields outside the core.
- **Fixtures:** the clause's Go violations, a variant and an error type that each have a pointer field, and a factory function that takes `*string` and returns a failure for nil.
- **Why no existing tool:** no linter restricts a field's type by where its struct is declared.

### DeclaredHook

- **Clauses:** `no-hooks-reached-through-catch-all-parameters`
- **Reports:** a method declared with the name and signature that standard-library code looks for on a value it took as `any`: `String`, `GoString`, `Format`, `LogValue`, `MarshalJSON`, `UnmarshalJSON`, `MarshalText` and `UnmarshalText`. Its message says to give the form a method with its own name, and to call it.
- **Allows:** `Error`, which `error` declares, and `ServeHTTP`, which `http.Handler` declares, since code reaches each through a parameter of that interface; a method with one of those names and another signature.
- **Fixtures:** the clause's Go violations, and an enum with a `Text()` method that its callers call.
- **Why no existing tool:** forbidigo bans a use of a name, not a method declared with one.

### EmbeddedField

- **Clauses:** `types-declare-every-method-they-have`
- **Reports:** an embedded field of a struct: a struct or a pointer to one, the project's or not, such as `*sql.DB` or `sync.Mutex`, and an interface.
- **Allows:** an interface that embeds another interface.
- **Fixtures:** the clause's four Go violations, and an interface that embeds `io.Reader`.
- **Why no existing tool:** golangci-lint's `embeddedstructfieldcheck` checks where an embedded field is placed, and no linter bans one.

### UnknownType

- **Clauses:** `unknown-types-stay-in-adapter-and-entrypoint-code`
- **Reports:** in packages under `core/` and `orchestrators/`, a parameter, result, field or variable declared as `any` or `interface{}`, or as a pointer, slice, array, map or channel of one, a named type with such an underlying type included.
- **Allows:** a type parameter and its constraint, such as `[T any]`; an interface with methods; the same declarations in adapter and entrypoint code.
- **Fixtures:** the clause's violations that Go can write, a generic function with `[T any]`, and a driven adapter that takes a `map[string]any`.
- **Why no existing tool:** the compiler makes code assert an `any` before using it, and lets one be declared anywhere.

### OpenInterfaceMatch

- **Clauses:** `open-interfaces-are-not-matched-by-implementation`
- **Reports:** a type switch, or a type assertion with one result or two, to a concrete type, on a value whose type is an interface that the project or a first-party dependency declares with no unexported method.
- **Allows:** a type switch on a sealed interface; a switch or an assertion on an `error`, and `errors.As`; one on an interface the standard library or a third party declares, such as `ast.Node`; one on an `any` in adapter and entrypoint code, which narrows it.
- **Fixtures:** the clause's violations, a type switch over a sealed interface, and `errors.As` with an error type.
- **Why no existing tool:** gochecksumtype checks a type switch on a sealed interface, and ignores every other.

### LocalReassignment

- **Clauses:** `core-locals-are-not-reassigned-once-read`
- **Reports:** in packages under `core/`, an assignment, a compound assignment, `++` or `--` to a local variable that some path has already read.
- **Allows:** a local assigned on several paths before any reads it; a loop-carried variable, assigned only before a loop and inside it; a variable that holds only an `error`, assigned again after the last was checked.
- **Fixtures:** the `unsupported` case's violations, and one function for each of the three allowances.
- **Why no existing tool:** ineffassign reports a value that is never read, not a local assigned after a read, and revive's `modifies-parameter` reports parameters only.

### FieldWrite

- **Clauses:** `fields-are-assigned-only-at-construction`, `built-values-are-not-changed`, `arguments-are-not-mutated`, `core-does-nothing-impure`, `orchestrator-code-performs-no-effects`
- **Reports:** outside packages under `adapters/`, an assignment, a compound assignment, `++` or `--` whose target is a field, reached through a pointer or not, or a pointer's target, as in `*p = v`; and a call of a method that writes through its receiver, found with SSA in the project's code and its dependencies'. A field written after construction is the held state that the two purity clauses' gaps name.
- **Allows:** a composite literal that sets its fields; writes in adapter code, whose roles may hold state, and in test files, where fakes are declared until fake code has a place of its own; a method that must match an interface declared outside the project and has to change its receiver; in entrypoint code, a call of a method of a standard-library or third-party value the entrypoint built, until it passes the value on or starts it.
- **Fixtures:** the violations of `fields-are-assigned-only-at-construction`, the purity clauses' cache kept in a field, the field and method violations of `arguments-are-not-mutated`, the third-party setters and the two entrypoint violations of `built-values-are-not-changed`, a driven adapter with a field it changes, and an entrypoint that registers its routes before it starts the server. A change to an element of a slice or a map is `CollectionWrite`'s.
- **Why no existing tool:** Go has no read-only field, and no linter reports a write to one; revive's `modifies-parameter` reports only a parameter assigned.
- **Specified in:** code-rulebook `docs/plans.md`, "Changes and call order", for `built-values-are-not-changed`.

### CollectionWrite

- **Clauses:** `shared-collections-are-not-changed`, `field-collections-are-not-changed`, `arguments-are-not-mutated`
- **Reports:** outside packages under `adapters/`, a change to a slice or a map (a write to an element, `append`, `delete`, `clear`, a `copy` into it or a sort in place) unless the function made it, with `make`, a literal or an `append` to nil, and hasn't yet shared it: passed it to a function other than those built-ins, stored it in a field, another collection or a pointer's target, captured it in a closure, sent it on a channel or returned it. So every change to a collection the function was passed, read from a field or got from a call is reported.
- **Allows:** filling a slice or a map the function made, before it shares it; changes in adapter code and in test files, as for `FieldWrite`; a method that must match an interface declared outside the project and has to fill what it is passed, such as `io.Reader`'s `Read`.
- **Fixtures:** the `unsupported` case's violations, the violations of `field-collections-are-not-changed`, the element, map, removal and sort violations of `arguments-are-not-mutated` and its buffer filled through an interface the project declares, a function that fills a slice it made and returns it, and an implementation of `io.Reader`.
- **Why no existing tool:** revive's `modifies-parameter` reports only an `append` assigned back to the slice parameter itself, and no linter follows whether a collection has been shared.
- **Specified in:** code-rulebook `docs/plans.md`, "Changes and call order", for `shared-collections-are-not-changed`.

### ImmutableType

- **Clauses:** `values-have-immutable-types`
- **Reports:** in packages under `core/` and `orchestrators/`, a parameter, result, field or local, declared or inferred, or a type argument of one, whose type is a standard-library or third-party type with a method that writes through its receiver, such as `*big.Int` or `strings.Builder`, or a pointer to a struct with exported fields, such as `*url.URL`.
- **Allows:** an interface, an error type and `context.Context`; a slice or a map, which is `CollectionWrite`'s; a struct held by value whose parts are all immutable types.
- **Specified in:** code-rulebook `docs/plans.md`, "Immutable types".

### IdentityComparison

- **Clauses:** `references-are-not-compared-by-identity`, `collections-are-not-keyed-by-instance`
- **Reports:** outside packages under `adapters/`: `==` and `!=` where both operands are pointers, or both are interfaces that a pointer type implements; a `switch` whose tag is one of those; `slices.Contains` and `slices.Index` on a slice of them; and a map type keyed by one.
- **Allows:** either operand `nil`; an enum's values, numbers, strings and structs held by value, which `==` compares by value; a sealed interface whose variants are all held by value; an `error` operand, which is errorlint's.
- **Fixtures:** both clauses' Go violations, a `*Order` compared with `nil`, and two value objects compared with `==`.
- **Why no existing tool:** errorlint reports `==` between errors only, and no linter reports a comparison or a map's key by its type.

### CoreTypeConstruction

- **Clauses:** `core-types-have-factory-functions`, `core-types-are-built-only-by-their-factory-functions`
- **Reports:** for a struct that a package under `core/` declares, an exported field, and no function in its package that returns it; and, outside a core type's package, each way of getting one without a factory function: `var t T`, `T{}`, `&T{}`, `new(T)`, a named result of type `T`, and a conversion to the type, such as `Kind(7)`.
- **Allows:** every use inside the type's own package, which is `OneConstructionSite`'s; `*T` and `nil` outside it.
- **Fixtures:** the `hidden-fields` case's violations except the panic, which is `NoPanic`'s; the second clause's violations that Go can write; and a driven adapter that returns the `*Order` a factory function gave it.
- **Why no existing tool:** the compiler stops another package setting an unexported field, and not from taking the zero value or converting to the type.
- **Specified in:** code-rulebook `rulebook/rules/design/constructors-only-store.yaml`, the `TODO(go-baseline)` comment, for the second clause.

### OneConstructionSite

- **Clauses:** `one-factory-function-calls-each-constructor`
- **Specified in:** code-rulebook `docs/plans.md`, "One construction site", which gives the Go analyzer.

### PortReturnType

- **Clauses:** `port-methods-return-no-interfaces`
- **Reports:** a method of an interface that a package under `orchestrators/` declares, with a result whose type is an interface.
- **Allows:** `error`; a sealed interface, which is a core type and not a handle.
- **Specified in:** code-rulebook `docs/plans.md`, "Changes and call order".

### PortMethodName

- **Clauses:** `port-methods-need-no-call-order`
- **Reports:** a method of an interface that a package under `orchestrators/` declares, named `Begin`, `Commit`, `Rollback`, `Open`, `Close`, `Init`, `Connect`, `Start`, `End` or `Flush`.
- **Allows:** every other name. A method that fails unless another was called first isn't in its name, and stays the gap.
- **Fixtures:** the clause's first, second, third and fifth violations, and a port with `ByID` and `Save` methods.
- **Why no existing tool:** forbidigo bans a use of a name, not a method declared with one.
- **Specified in:** code-rulebook `docs/plans.md`, "Changes and call order".

### OrchestratorLoop

- **Clauses:** `orchestrator-code-runs-no-loop-until-stopped`
- **Reports:** in packages under `orchestrators/`: a `for` with no condition, unless a `break` or a `return` inside it is guarded by what a function in the core returned; a `for` whose condition calls a method of an interface or tests a context, as in `ctx.Err() == nil`; and a `range` over a channel.
- **Allows:** a `range` over a slice, a map or an integer; a loop that ends on what a function in the core returned, such as reading pages until the last.
- **Fixtures:** the clause's violations, and an orchestrator that reads pages until the core says it has them all. A loop over pieces of work that could each fail alone isn't in the code, and stays the gap.
- **Why no existing tool:** no linter reports a loop by what ends it.

### DeadlineWait

- **Clauses:** `nothing-sleeps`
- **Reports:** a receive that waits for nothing but time: from `time.After`, from a timer's or a ticker's channel, or from the `Done` channel of a context the function itself gave a deadline, as a statement or an expression of its own, or in a `select` whose every case is one of them.
- **Allows:** a `select` that also waits on a channel of work, a signal or a context the function was passed.
- **Fixtures:** the clause's third and fifth violations, and a `select` on a timer and a channel of messages. `time.Sleep` and the timers themselves are forbidigo's, and a third-party function that sleeps is the dependency list's.
- **Why no existing tool:** forbidigo bans a call by its name, and a receive from a channel has none.

### AdapterBoundary

- **Clauses:** `driving-adapter-code-calls-no-driven-adapter`, `driven-adapter-code-calls-no-orchestrator`
- **Reports:** a value whose type a package under `adapters/driven/` declares, given to driving-adapter code: passed as an argument to a function or method of a package under `adapters/driving/`, or stored in a field of one of its structs. In the same way, a value whose type a package under `orchestrators/` declares, given to driven-adapter code. It finds each as `CoreBoundary` finds what is passed to the core, with an interface, a function value or a closure that holds one included.
- **Allows:** an orchestrator given to a driving adapter; a driven adapter given to an orchestrator; the system adapters the vocabulary says driving-adapter code also calls, the stopwatch and the deadlines, by their full names in go-shared-code.
- **Fixtures:** both clauses' violations, the controller given the system clock among them, and a driving adapter given the system stopwatch.
- **Why no existing tool:** depguard reports an import, and a driving adapter that declares an interface of its own imports nothing to be given a driven adapter through it.

### ErrorMatch

- **Clauses:** `error-types-of-other-roles-are-not-matched`
- **Reports:** an `errors.Is` or `errors.As` call, a type switch or a type assertion on an `error`, or a comparison of one, whose target is an error value or an error type that a package of another role declares, in the project or a first-party dependency, unless that role is the core.
- **Allows:** a target that the matching code's own role or the core declares; one that the standard library or a third party declares, such as `sql.ErrNoRows`.
- **Fixtures:** the clause's violations, a driving adapter that matches an error its orchestrator's package declares, and a driving adapter that matches an error the core declares.
- **Why no existing tool:** depguard stops only an import, and a driving adapter may import the orchestrator package whose error it matches, as `cmd/` may import every role's.

### SuppressionProblems

- **Clauses:** `checks-are-not-suppressed`, `visibility-is-not-bypassed`
- **Specified in:** code-rulebook `docs/plans.md`, "Checks at full strength", the suppression-comment scan, which gives Go's comments and build constraints, and "Escape hatches", which adds `//go:linkname`.

### ConfigFileProblems

- **Clauses:** `baseline-settings-are-not-weakened`, `checks-are-not-suppressed`, `check-task-runs-every-baseline-check`, `check-fails-on-unformatted-files`, `formatter-settings-come-from-the-baseline`
- **Reports:** a `.golangci.json`, `.golangci.toml` or `.golangci.yaml` at the project's root, each of which golangci-lint reads ahead of the `.golangci.yml` that `verify` compares.
- **Allows:** `.golangci.yml` as the baseline generates it.
- **Fixtures:** a project with each of the three files beside a generated `.golangci.yml`, and a project with `.golangci.yml` alone.
- **Why no existing tool:** golangci-lint has no setting that refuses a second configuration file, and one would be in the file a project can replace.
- **Specified in:** code-rulebook `docs/plans.md`, "Checks at full strength", the effective-settings comparison, whose other reports `verify` already makes by comparing `.golangci.yml` byte for byte.

### CoreReachesNothingImpure

- **Clauses:** `core-calls-reach-nothing-impure`, `core-loading-does-nothing-impure`
- **Fixtures:** testdata modules for a pure third-party package the core calls, which must pass; one whose function calls `time.Now` two packages deep; one that prints with `fmt.Println`; one whose function writes a package-level map it keeps as a cache; one whose `init` reads an environment variable, which the core imports and never calls; one that imports `os` but whose function the core calls reaches no `os` call, which must pass; one that calls a method on an interface value it builds itself, whose implementation reads the clock; and one that calls a function value it was passed, which must pass.
- **Needs first:** no new driver, since facts already reach every dependency: `analyse` gives `All` to `gopackages.Analyze`, which loads with `packages.LoadAllSyntax` and calls `checker.Analyze`, and `checker` runs an analyzer that declares `FactTypes` on every package the roots import, at any depth, third-party and standard-library packages included, as `analysistest.Run` does in the tests. Four things to decide: whether a standard-library function is a leaf, judged only by the map of impure operations, or followed into its body, where `unsafe`, `sync.Pool` and the runtime would mark `fmt.Sprintf` impure, and with that whether `os` in the map takes in `os/user` and `os/signal`; what a dynamic call on a value the code built itself reaches when SSA can't resolve it, since a fact-based pass sees only the packages below it and not every implementation in the program, so it is either reported as unresolved or needs a whole-program pass as `RoleProblems` does; whether every run analyses every dependency again until verdicts are cached by module version and `go.sum` entry; and which of the clause and the plan to follow where they differ: `core-calls-reach-nothing-impure`'s fourth violation counts an implementation the code didn't build, where the plan resolves the one it built.
- **Specified in:** code-rulebook `docs/plans.md`, "Purity of what the core reaches", "The method" and "Go", which replaces `verify-dependencies`' import-following for these two clauses.

## Waiting on a decision

### CatchAll

- **Clauses:** `driving-adapter-code-catches-everything-only-around-invocations`
- **Reports:** a call of `recover` in a package under `adapters/driving/`, except in the one function the project marks as its invocation catch-all.
- **Allows:** the marked catch-all. `recover` in the core, in orchestrators and in driven adapters is forbidigo's, by path.
- **Fixtures:** the clause's violations, and a marked catch-all.
- **Why no existing tool:** forbidigo's `^recover$` can't allow one function.
- **Needs first:** how a project marks its invocation catch-all, such as a name the baseline fixes for it.
- **Specified in:** code-rulebook `TO-IMPLEMENT.md`, "Errors: catching and throwing by role".

### InvocationCatchAll

- **Clauses:** `driving-adapter-catches-each-invocation`
- **Reports:** in packages under `adapters/driving/`: an `http.Server` built with, or `http.ListenAndServe` or `http.Serve` called with, a handler that isn't what the project's catch-all returned; a `go` statement whose function doesn't defer the catch-all; and a function given to go-shared-code's system timer that doesn't run each pass inside it.
- **Allows:** a command line, whose only invocation is the whole run. A consumer's loop over messages, and another framework's start of an invocation, aren't found, and stay the gap.
- **Fixtures:** the clause's second, third and fourth violations, and a server whose handler is wrapped.
- **Why no existing tool:** no linter knows which function is the catch-all, or where an invocation starts.
- **Needs first:** how a project marks its invocation catch-all, as for `CatchAll`; and go-shared-code's system timer, for the last report.
- **Specified in:** code-rulebook `TO-IMPLEMENT.md`, "Errors: catching and throwing by role", which expects `partial` for one framework.

### ConstructorBody

- **Clauses:** `constructors-only-store-their-arguments`
- **Reports:** a constructor that does anything but store each of its parameters, unchanged, in a field. If Go's constructor is the composite literal, that is a literal of a struct the project declares with a field's value that is anything but a parameter or a local, such as a call or a conversion. If it is the function that returns a service, that is one whose body does more than return the literal, or that returns an `error` beside the instance.
- **Allows:** a core type's factory function, which checks its invariants before the literal and returns `(T, error)`.
- **Fixtures:** the clause's violations that Go can write, a `NewStore(db)` that returns `&Store{db: db}`, and a factory function that checks its argument and then writes the literal.
- **Why no existing tool:** nothing tells a constructor from any other function.
- **Needs first:** what Go's constructor is: the composite literal, or the function that returns a service, which needs the role classifier's shapes to tell a service from a core type.

### FakeProblems

- **Clauses:** `every-port-has-one-fake`
- **Reports:** for each port, no type, or more than one, in fake code that implements it, found with `types.Implements` across packages; and a fake whose name isn't the port's with the affix.
- **Allows:** one fake for each port, named for it.
- **Fixtures:** the clause's violations, and a port with its one fake.
- **Why no existing tool:** no linter relates an interface to the types that implement it in other packages.
- **Needs first:** where fake code lives, which has to be a package that isn't a `_test` package, since the tests of several packages import a fake; and the affix a fake's name adds to its port's.
- **Specified in:** code-rulebook `TO-IMPLEMENT.md`, "Testing: fakes, contract tests, single calls and mutation testing".

### PortStandIn

- **Clauses:** `only-driven-adapters-and-fakes-stand-in-for-ports`, `no-hand-written-scripted-doubles`
- **Reports:** an argument or an assignment whose target type is a port and whose value's type is neither a driven adapter's nor the port's fake's, a struct that wraps the fake among them.
- **Allows:** a driven adapter, and the port's fake.
- **Fixtures:** the first clause's Go violation and its wrapper of a fake, the second clause's three violations, and a test that builds the port's fake.
- **Why no existing tool:** depguard reports imports, and no linter reads what a value passed as an interface is.
- **Needs first:** where fake code lives, as for `FakeProblems`.
- **Specified in:** code-rulebook `TO-IMPLEMENT.md`, "Testing: fakes, contract tests, single calls and mutation testing".

### ContractTestProblems

- **Clauses:** `every-port-has-one-set-of-contract-tests`, `fake-and-adapter-tests-run-the-contract-tests`
- **Reports:** a port with no contract tests at the fixed location, or with two sets; contract tests that reference a fake or a driven adapter; and the tests of a fake or of a driven adapter that don't call every one of its port's contract tests.
- **Allows:** contract tests that take a function that builds an implementation, called from the fake's tests and from each adapter's.
- **Fixtures:** both clauses' violations, and a port whose fake's tests and adapter's tests each run its contract tests.
- **Why no existing tool:** no linter relates a test to a port.
- **Needs first:** where a port's contract tests live and what they are called, from the port's name; and where fake code lives, as for `FakeProblems`.
- **Specified in:** code-rulebook `TO-IMPLEMENT.md`, "Testing: fakes, contract tests, single calls and mutation testing".

### OrchestratorTestCall

- **Clauses:** `orchestrator-test-case-calls-one-use-case-once`
- **Reports:** a test function, or a subtest's function, in the tests of a package under `orchestrators/` that makes no call, or more than one, to an exported function or method of that package, or makes one inside a loop.
- **Allows:** any number of calls on fakes and on the core.
- **Fixtures:** the clause's violations, and a test that sets a fake, calls once and checks the fake.
- **Why no existing tool:** nothing counts a test's calls by what they are calls to.
- **Needs first:** whether a table test's loop of `t.Run` subtests is one test case for each row, and whether a call made through a helper function counts.
- **Specified in:** code-rulebook `TO-IMPLEMENT.md`, "Testing: fakes, contract tests, single calls and mutation testing".

## Waiting on the role classifier

### RoleProblems

- **Clauses:** `every-declaration-has-one-role`, `a-unit-of-code-holds-one-role`, `every-type-has-one-shape`, `dependency-declarations-keep-their-role`
- **Specified in:** code-rulebook `docs/plans.md`, "Roles in each baseline", which gives its tests in order, its fixtures and what it reports. It is one `verify` step over the whole program, with `go/packages` and `go/types`, since an analyzer's facts don't pass from an adapter to the orchestrator whose port it implements. It takes the place of `RoleOf` and `RolelessProblems`, and the checks below take each declaration's role from it.

### RoleReference

- **Clauses:** `driven-adapter-code-references-only-ports-and-values`, `value-code-references-only-value-code`, `driving-adapter-code-references-only-orchestrators-and-the-core`, `entrypoint-code-calls-only-driving-adapters-and-values`, `only-adapter-and-entrypoint-code-references-plain-data-types`, `port-units-of-code-follow-interface-implementation`
- **Reports:** a reference or a call from one declaration to another whose role the vocabulary's `may-reference` and `may-call` don't allow the first: decision code named from value code or from driven-adapter code; a port named from driving-adapter code; a call from entrypoint code to an orchestrator, a driven adapter or decision code, other than to build, open or close one; a plain-data type named outside adapter and entrypoint code; and a port declared in a package other than its orchestrator's.
- **Allows:** every reference and call the vocabulary lists; entrypoint code building and closing what it wires; a driving adapter's reference to a system adapter the vocabulary says it also calls.
- **Fixtures:** the clauses' violations, the vocabulary's example for each allowed reference, and the conformance fixtures that depguard's rules pass and fail today, since this check replaces them.
- **Why no existing tool:** depguard's rules go by a package's path, and can't tell a decision from a value or a port from the orchestrator declared beside it.
- **Needs first:** `RoleProblems`.

### SignatureType

- **Clauses:** `port-methods-take-and-return-signature-types`, `orchestrator-takes-and-returns-signature-types`, `decisions-driving-adapters-call-take-and-return-signature-types`, `factory-functions-take-signature-types`, `orchestrator-is-given-ports`, `the-core-takes-no-plain-data`
- **Reports:** a parameter or result that isn't a signature type, on a port's method, an orchestrator's use-case function or method, a decision function or method that driving-adapter code calls, and a parameter of a core type's factory function; and a collaborator of an orchestrator, held in a field or passed to a use-case function, that is none of a port, another orchestrator and a signature type, such as a `*slog.Logger`, an `io.Writer` or an interface the orchestrator didn't declare.
- **Allows:** a core type; an integer, a float and a `string`; `bool` as a result; a pointer to one or a slice of one; `error` and `context.Context`; no result.
- **Fixtures:** the six clauses' violations that Go can write, and a port, an orchestrator and a factory function that take and return core types and scalars.
- **Why no existing tool:** the type checker knows a type, and not whether it is a core type or which role's code calls a function.
- **Needs first:** `RoleProblems`, for which types are core types, which interfaces are ports and which functions driving-adapter code calls.

### DecisionResultFlow

- **Clauses:** `driving-adapter-code-only-reports-what-a-decision-returns`
- **Reports:** in driving-adapter code, a value that a call to decision code returned, passed to an orchestrator or to another call to decision code, or read by a condition whose branch calls an orchestrator. It follows the value as `PortBranch` follows one read through a port.
- **Allows:** building the response, the output or the exit code from it.
- **Fixtures:** the clause's violations, and a handler that calls a decision and renders what it returned.
- **Why no existing tool:** no linter follows a value from one call to another by the roles of the code called.
- **Needs first:** `RoleProblems`, to tell decision code from value code.

### DerivedBranch

- **Clauses:** `orchestrator-code-branches-only-on-decisions`
- **Reports:** in orchestrator code, a condition that depends on a value that a core type's method derived from one read through a port, such as whether a list of orders is empty. `PortBranch` reports a condition on the port's value itself and takes every call into the core to clear it; this check keeps the value marked through a call of value code.
- **Allows:** a condition on what decision code returned; a test of whether the call failed, as `err != nil` or a nil pointer.
- **Fixtures:** the clause's fourth and fifth violations, and an orchestrator that passes what it read to a decision and branches on the answer.
- **Why no existing tool:** `PortBranch` can't tell a core type's method from a decision function.
- **Needs first:** `RoleProblems`, to tell value code from decision code.

## Waiting on something else

### LoopShape

- **Clauses:** `loops-compute-no-value-an-operation-names`
- **Needs first:** Go's shared collection operations, which the loops are replaced with.
- **Specified in:** code-rulebook `docs/plans.md`, "Loop-shape check".

### CollectionCallback

- **Clauses:** `collection-operations-take-pure-functions`, `folds-compute-no-value-a-narrower-operation-names`
- **Reports:** a function passed to a shared collection operation that calls a method of a port, an orchestrator or an adapter, performs a banned effect, or assigns to a variable it captured; and a function passed to the shared fold whose body only appends, filters, adds, groups, keeps a least or greatest element, or sets a boolean.
- **Allows:** a pure function; a fold whose step reads the value so far, such as a running balance.
- **Fixtures:** both clauses' violations that Go can write, and a fold that merges overlapping intervals.
- **Why no existing tool:** forbidigo bans a call by its name, and not what a function passed to another does.
- **Needs first:** Go's shared collection operations.
- **Specified in:** code-rulebook `docs/plans.md`, "Collection operations", which names the candidates.

### OwnCollectionOperation

- **Clauses:** `collection-operations-come-from-one-place`
- **Reports:** a generic function the project declares that takes a slice or a map of a type parameter with a function, or returns a slice or a map of one; and a named type over a slice or a map, or a struct that wraps one, with a method that takes a function.
- **Allows:** the shared collection operations' own code, by its full name; a function over one concrete type's slice.
- **Fixtures:** the clause's first violation, a type that wraps a slice with `Map` and `Filter` methods, and a call of a shared operation. A utility library is depguard's.
- **Why no existing tool:** depguard bans a library by its path, not a function by its shape.
- **Needs first:** Go's shared collection operations, which replace what it reports.
- **Specified in:** code-rulebook `docs/plans.md`, "Collection operations", which names the candidates.

### InternalValue

- **Clauses:** `outside-facing-code-names-no-internal-type`, `external-driving-adapters-get-no-internal-values`, `external-driving-adapters-get-no-internal-values-from-decisions`, `external-driven-adapters-take-no-internal-values`, `only-decisions-and-internal-adapters-read-internal-values`
- **Needs first:** go-shared-code's shared ID package, with the Internal wrapper, and the baseline's capability list, which says whether an adapter is internal or external.
- **Specified in:** code-rulebook `docs/plans.md`, "Identity and the schema", the internal-values checks: a reference ban, a crossing check and a call-site check, which may be three analyzers.

### IDType

- **Clauses:** `ids-are-held-as-their-id-types`, `each-public-id-has-a-type`
- **Reports:** a field, parameter or result that holds a row's ID or public ID as a `string`, as bytes or as a third-party UUID type, and an ID field that is a pointer.
- **Allows:** an ID type built on the shared ID package.
- **Fixtures:** both clauses' violations that are in code and not in a schema.
- **Why no existing tool:** no linter knows which strings are IDs.
- **Needs first:** go-shared-code's shared ID package, and a way to tell which values are IDs, such as a field named `id` or ending `ID`.
- **Specified in:** code-rulebook `docs/plans.md`, "Identity and the schema", the static checks.

### ListedDependencyProblems

- **Clauses:** `every-third-party-dependency-is-listed`, `at-most-one-listed-dependency-per-capability`, `orchestrator-code-reaches-nothing-impure`, `adapters-and-entrypoints-reach-only-their-roles-operations`
- **Reports:** from `go list -deps`: a third-party package the project's code imports whose module the baseline's dependency list doesn't have; one that the list gives an impure operation the importing package's role may not perform, at any depth of import; and two entries of one capability that the project uses, the standard library's entries such as `log` and `log/slog` included.
- **Allows:** a listed module whose operations the importing role may perform; a standard-library package the list doesn't name.
- **Fixtures:** the four clauses' violations that Go can write, and an orchestrator importing a listed module whose entry gives it no impure operation. What the core reaches is `CoreReachesNothingImpure`'s, judged from the code rather than from the list.
- **Why no existing tool:** `DependencyProblems` asks only whether a third-party package of the core or an orchestrator reaches an I/O package, not which operation it performs or whether anyone vetted it, and depguard's lists have no capabilities.
- **Needs first:** the baseline's dependency list and capability list, in the form code-rulebook's `docs/baselines.md` gives.

### SourceField

- **Clauses:** `standard-library-and-third-party-code-takes-sources-from-the-system-adapters`
- **Reports:** a composite literal of a type that takes a source of the time or of randomness and leaves that field unset, such as a `tls.Config` without `Time` or `Rand`; and a value of a listed third-party type built without the option its dependency-list entry names under Sources.
- **Allows:** a literal that sets each such field. Whether what it is given asks the system adapter isn't checked, and stays the gap.
- **Fixtures:** the clause's violations, and a `tls.Config` with both fields set.
- **Why no existing tool:** no linter requires one field to be set in a literal of one type.
- **Needs first:** go-shared-code's system clock and system random source, which are what code passes; and the baseline's dependency list, for the third-party types.
