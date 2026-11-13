# go-baseline

The settings every Go project shares, and the command that keeps a project on them. It is the Go counterpart of `@ianflanagan1/ts-baseline`; new projects start from [go-template](../go-template).

## Starting a project

From this repo, for an application:

```sh
devbox run -- task new MODULE=github.com/ianflanagan1/<name> DIR=~/projects-ianf/<category>/<name>
```

or, for a library, `task new-library` with the same variables.

The directory must be an absolute path, because devbox runs tasks from this repo's root. The task, in `Taskfile.project.yml`:

1. copies go-template with `gonew`, rewriting the module path in `go.mod` and every import;
2. for an application, renames `cmd/app` to `cmd/<name>`; for a library, removes `cmd/` and go-template's `Taskfile.project.yml`, with its `build` task, and adds `core/<package>/doc.go`, its package named for the module without a `go-` prefix;
3. writes a README skeleton, and removes the licence, so you choose one;
4. upgrades go-baseline to its latest release, so a project never starts on an old baseline;
5. generates `.golangci.yml`, `BASELINE.md`, the CI workflow and `Taskfile.yml` for the new module;
6. runs `devbox run -- task check`;
7. creates a git repo on `main` with one commit.

Publish it with `github publish` from repo-tools.

## What a project gets

| Setting | Where | Checked by |
|---|---|---|
| golangci-lint config, generated from the module path in `go.mod` | `.golangci.yml` | `verify`: must match byte for byte |
| CI: `devbox run -- task check` on each push and pull request | `.github/workflows/check.yml` | `verify`: must match byte for byte |
| `check`, `lint`, `lint-file`, `test` and `integration` tasks, including the project's own from `Taskfile.project.yml` | `Taskfile.yml` | `verify`: must match byte for byte |
| The architecture and conventions, with examples, for people and AI agents | `BASELINE.md`, imported by the project's `CLAUDE.md` | `verify`: must match byte for byte |
| Go pinned to an exact version, golangci-lint and go-task to the baseline's versions | `devbox.json` packages | `verify` |
| `go.mod`'s `go` line is the minor version of the Go that devbox pins | `go.mod` | `verify` |
| `GOTOOLCHAIN=local` | `devbox.json` env | `verify` |
| [coupled-tags-lint](https://github.com/ianflanagan1/coupled-tags), which `check` runs to lint the `COUPLED:` tags | `devbox.json` packages, with the commit pinned in `devbox.lock` | `verify` |
| No scripts: every command is a task | `devbox.json` | `verify` |
| `cache/`, where tools keep their caches, is ignored with the line `/cache/` | `.gitignore` | `verify` |
| Values reaching the core through interfaces, functions, fields or type arguments are pure (`coreboundary`) | code | `verify` |
| Configuration (environment, flags, `os.Args`) is read only in `cmd/` (`configread`) | code | `verify` |
| No third-party dependency of a core or orchestrator package imports an I/O package, at any depth | `go.mod` | `verify`, from `go list -deps` |
| `go.mod` and `go.sum` are tidy | `go.mod` | `verify`, with `go mod tidy -diff` |
| No call reaches a function with a known vulnerability | code and `go.mod` | `verify`, with govulncheck |
| In a module with a program in `cmd/`, every function under `cmd/` and `internal/` is reachable from one | code | `verify`, with deadcode |
| GitHub Actions pinned by commit SHA | `.github/workflows/check.yml` | `verify`: must match byte for byte |
| Each core package's tests cover at least 80% of its statements | tests | `test` |
| No data races in the tests | tests | `test`, with `-race` |
| No `src/` or `pkg/` at the root, no `util`, `common` or `helpers` package | directories | `verify` |

`devbox run -- task check` runs `go tool go-baseline verify`, then the linters, then `go tool go-baseline test`.

## The layout the linters enforce

[BASELINE.md](BASELINE.md) sets out the roles (core, orchestrator, driving adapter, driven adapter, entrypoint), the rules for each and examples. Its source is `internal/core/baseline/BASELINE.md`. A package's directory decides its role: `RoleOf` in `baseline.go` and the paths in the golangci-lint template must agree. The analyzers use `RoleOf` too. depguard enforces the import rules; forbidigo bans the clock, printing and the global loggers outside `cmd/` and adapters; revive's modifies-parameter, at confidence 0.5, reports a parameter reassigned; gocognit keeps every function but `cmd/`'s simple enough to review, and funlen keeps adapters thin and `cmd/` small. Test files are exempt from both, so integration tests can use real dependencies. The other linters cover the Go guide: exhaustive switches (`exhaustive`, `gochecksumtype`), no package-level state (`gochecknoglobals`, `gochecknoinits`), doc comments on exported names (`revive`), `errors.Is` and `errors.As` (`errorlint`), errors from other modules wrapped where they come in (`wrapcheck`) and `package foo_test` tests (`testpackage`). gosec, modernize and govet's nilness look for bugs and risky code. gosec's G204 and G304, running a command or opening a file named by a variable, are allowed in `cmd/` and adapters, whose job that is, and so is G703, which reports writing data read from a file as path traversal even when the path is a constant. gofumpt and goimports format.

## Tools considered

Decisions on the common Go analysis tools, recorded 28 October 2026. golangci-lint's `standard` set is errcheck, govet, ineffassign, staticcheck and unused.

### In use

| Tool | How |
|---|---|
| Compiler, `go vet`, `internal/` | Toolchain and language. `go vet` runs as golangci-lint's govet |
| staticcheck, U1000 | golangci-lint's `staticcheck` and `unused` |
| errcheck, errorlint, exhaustive, gochecksumtype, depguard, revive | golangci-lint |
| gocognit | Cognitive complexity at most 10, everywhere but `cmd/` and tests |
| funlen | golangci-lint, for adapters and `cmd/` |
| revive and staticcheck ST | Both. revive's `exported` rule enforces doc comments |
| goimports | golangci-lint formatter |
| golangci-lint | Pinned in `devbox.json`; `verify` checks the version |
| gofumpt | golangci-lint formatter, in place of gofmt |
| wrapcheck | golangci-lint. Errors from the module's own packages, and in tests, are exempt: they already carry context |
| `go mod tidy -diff` | `verify` fails when `go.mod` or `go.sum` is untidy |
| `go test -race` | In `go tool go-baseline test` |
| govulncheck | In `verify`, through `golang.org/x/vuln/scan`, so its version is pinned in go-baseline's `go.mod` |
| modernize | golangci-lint; Go is pinned, so newer idioms are always available |
| nilness | govet analyser, which golangci-lint leaves off by default |
| deadcode | In `verify`, for modules with a program in `cmd/`, and only under `cmd/` and `internal/`: other modules may use any other package. U1000 can't see that an exported function is unused. It runs from the x/tools that go-baseline's `go.mod` pins |
| gosec | golangci-lint. G204, G304 and G703 are excluded in `cmd/` and adapters, and gosec in tests, in the template rather than with `//nolint` |

### Not adopted

| Tool | Why |
|---|---|
| gofmt | Replaced by gofumpt, a strict superset |
| golint | Deprecated; revive replaces it |
| gocyclo | Counts each `case`, so an exhaustive switch over a large enum or sum type fails it however simple the switch is, and the no-`//nolint` rule leaves no way out. gocognit counts a switch once |
| cyclop | Cyclomatic complexity, like gocyclo |
| revive's function-length rule | Duplicates funlen |
| gomodguard | `verify`'s dependency check, from `go list -deps`, is stricter |
| go-arch-lint, arch-go | depguard, `RoleOf` and the coreboundary, portbranch and configread analysers enforce the architecture more precisely |
| gopls modernize | Covered by golangci-lint's modernize |

### Open

| Tool | Question |
|---|---|
| nilaway | Runs outside golangci-lint and reports false positives, which the no-`//nolint` rule would force code to work around |
| dupl | Whether duplicated code is worth a threshold, given table-driven tests |
| goleak | Needs `goleak.VerifyTestMain` in each package, which `verify` would have to check |
| Semgrep or CodeQL | Whether deeper security analysis is wanted beyond gosec, and as a separate workflow |

## Adding an architecture rule

Rules that need a project's roles or type information across packages are analyzers, built on `golang.org/x/tools/go/analysis`, in `internal/core/analyzers`. To add one:

1. Write a function that returns an `*analysis.Analyzer` for an `analyzers.Project`, and add it to `All`.
2. Add sample code to `internal/core/analyzers/testdata/src`, marking each line that should be reported with a ``// want `regexp` `` comment, and test it with `analysistest.Run`.
3. Describe the rule in `BASELINE.md`, so agents meet it before the analyzer does.

`verify` runs every analyzer over the project, through the adapter in `internal/adapters/driven/gopackages`.

An analyzer or `verify` step the baseline still needs has an entry in [docs/analyzers-and-verify-steps-to-build.md](docs/analyzers-and-verify-steps-to-build.md). When a clause needs one the baseline doesn't have, or a requirement for a planned one arrives or changes, add or change its entry in the same change, and don't build the check in that change. Build one check to a session, test-first, and delete its entry in the change that adds it.

golangci-lint can't extend a shared config, so a project can't add its own rules to `.golangci.yml`. When one needs to, add the option to the baseline.

## The code rulebook

The [code rulebook](https://github.com/ianflanagan1/code-rulebook) states the rules every language shares, split into clauses. `internal/core/rulebook` records which of go-baseline's checks enforce each clause and the tests that prove it. `rulebook-coverage.json` holds that listing. One test fails when the file is out of date or names a test that doesn't exist; another, `TestTheListingFitsThePinnedRulebook`, runs the pinned release of the rulebook's `check-listing` over it, which fails on a clause that applies to Go and isn't assessed, a role that isn't mapped, or a claim of full or partial coverage without proof.

code-rulebook is a private repository, so that test needs Git access to it. Locally, have Git fetch it over SSH: `git config --global url."git@github.com:ianflanagan1/code-rulebook".insteadOf "https://github.com/ianflanagan1/code-rulebook"`. In CI, give the job a token that can read it.

[docs/analyzers-and-verify-steps-to-build.md](docs/analyzers-and-verify-steps-to-build.md) lists the analyzers and `verify` steps still to build, each with the clauses it enforces, what it must report and allow, and its fixtures. Each is built alone, test-first, and its entry deleted when it is. The tests in `internal/core/rulebook/checkstobuild_test.go` fail when the list is empty, or an entry is incomplete, names a clause the listing lacks or covers in full, or names a check that exists.

`TestConformance` in `internal/core/baseline` is the proof for the linters. It runs golangci-lint, with the config `write` generates, over `testdata/conformance`, which has a package for each violation the rulebook describes. Each subtest is named for its clause. `TestGoHasNoSyntaxFor`, `TestTheCompilerRejects`, `TestGoHasNoBuiltinThatRunsText`, `TestAnInterfaceWithAnUnexportedMethodIsImplementedOnlyInItsPackage` and the tests that build the other modules under `testdata` prove the claims Go itself enforces.

When a check changes what it enforces, or the baseline moves to a new rulebook release:

1. Bump `rulebookRelease` in `internal/core/rulebook/rulebook_test.go`, if moving.
2. Assess each clause the test reports. Where an existing check reaches, add the violation to `testdata/conformance` and its expectation to `TestConformance`, or write another test that shows the check catching it. Where catching it needs a check go-baseline doesn't have, such as a new analyzer, record the coverage the existing checks give and write down what the missing check must do; it is built separately. A missing analyzer or `verify` step gets an entry in [docs/analyzers-and-verify-steps-to-build.md](docs/analyzers-and-verify-steps-to-build.md).
3. Update the claim in `internal/core/rulebook/rulebook.go`.
4. `devbox run -- task rulebook-coverage`, then `check`.

## Changing a shared setting

1. Change it here: the template in `internal/core/baseline/golangci.yml.tmpl`, the guide in `internal/core/baseline/BASELINE.md`, the tasks in `internal/core/baseline/Taskfile.yml`, or the constants in `baseline.go`. To let the core and orchestrators use a third-party package the dependency check rejects, add it to `dependencyAllowlist` with the reason, after reading the code projects call. When bumping golangci-lint or go-task, change `GolangciLintVersion` or `TaskVersion` and the version in this repo's `devbox.json` together; this repo's own `verify` fails until they agree. Pin a GitHub Action in `check.yml` by its commit SHA, with the tag in a comment.
2. `devbox run -- task check`, commit, then `github release . minor`.
3. In go-template and each project:

   ```sh
   devbox run -- go get -tool github.com/ianflanagan1/go-baseline/cmd/go-baseline@latest
   devbox run -- go tool go-baseline write
   devbox run -- task check
   ```

   A project that hasn't done this keeps passing on the old version; one that upgrades without `write` fails `verify`.

## Commands

| Command | What it does |
|---|---|
| `go tool go-baseline write` | Writes `.golangci.yml` for the module in `go.mod`, `BASELINE.md` and `.github/workflows/check.yml` |
| `go tool go-baseline verify` | Lists each way the project differs from the baseline, including I/O dependencies of the core and orchestrators, packages with no role and the analyzers' findings, and fails if there are any |
| `go tool go-baseline test` | Runs the tests with coverage, and fails if a core package is below 80% |
| `go tool go-baseline map` | Lists each package's role, the ports the orchestrators declare and the driven adapters that implement them |
| `go tool go-baseline rulebook` | Prints how the baseline enforces each clause of the [code rulebook](https://github.com/ianflanagan1/code-rulebook), as the coverage listing the rulebook checks |
