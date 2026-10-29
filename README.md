# go-baseline

The settings every Go project shares, and the command that keeps a project on them. It is the Go counterpart of `@ianflanagan1/ts-baseline`; new projects start from [go-template](../go-template).

## Starting a project

From this repo, with the module kind: `command`, `pure-library` (no I/O; every package is core) or `adapters` (shared adapters, such as go-adapters):

```sh
devbox run new <kind> github.com/ianflanagan1/<name> ~/projects-ianf/<category>/<name>
```

The directory must be an absolute path, because devbox runs scripts from this repo's root. The script:

1. copies go-template with `gonew`, rewriting the module path in `go.mod` and every import;
2. for a command, renames `cmd/app` to `cmd/<name>`; otherwise removes `cmd/` and adds a root `doc.go`, its package named for the module without a `go-` prefix;
3. writes a README skeleton for the kind, and removes the licence, so you choose one;
4. upgrades go-baseline to its latest release, so a project never starts on an old baseline;
5. generates `.golangci.yml`, which records the module kind, `BASELINE.md` and the CI workflow for the new module;
6. runs `devbox run check`;
7. creates a git repo on `main` with one commit.

Publish it with `github publish` from repo-tools.

## What a project gets

| Setting | Where | Checked by |
|---|---|---|
| golangci-lint config, generated from the module path in `go.mod` | `.golangci.yml` | `verify`: must match byte for byte |
| CI: `devbox run check` on each push and pull request | `.github/workflows/check.yml` | `verify`: must match byte for byte |
| The architecture and conventions, with examples, for people and AI agents | `BASELINE.md`, imported by the project's `CLAUDE.md` | `verify`: must match byte for byte |
| Go pinned to an exact version, golangci-lint pinned to the baseline's version | `devbox.json` packages | `verify` |
| `GOTOOLCHAIN=local` | `devbox.json` env | `verify` |
| `lint`, `test`, `integration` and `check` scripts | `devbox.json` scripts | `verify` |
| Values reaching the core through interfaces, functions, fields or type arguments are pure (`coreboundary`) | code | `verify` |
| Configuration (environment, flags, `os.Args`) is read only in `cmd/` (`configread`) | code | `verify` |
| No third-party dependency of a core or shell package imports an I/O package, at any depth | `go.mod` | `verify`, from `go list -deps` |
| `go.mod` and `go.sum` are tidy | `go.mod` | `verify`, with `go mod tidy -diff` |
| No call reaches a function with a known vulnerability | code and `go.mod` | `verify`, with govulncheck |
| Each core package's tests cover at least 80% of its statements | tests | `test` |
| No data races in the tests | tests | `test`, with `-race` |
| No `src/` or `pkg/` at the root, no `util`, `common` or `helpers` package | directories | `verify` |

`devbox run check` runs `go tool go-baseline verify`, then the linters, then `go tool go-baseline test`.

## The layout the linters enforce

[BASELINE.md](BASELINE.md) sets out the roles (core, shell, adapter, composition root), the rules for each and examples. Its source is `internal/core/baseline/BASELINE.md`. A package's directory decides its role: `RoleOf` in `baseline.go` and the paths in the golangci-lint template must agree. The analyzers use `RoleOf` too. depguard enforces the import rules; forbidigo bans the clock, printing and the global loggers outside `cmd/` and adapters; gocyclo, gocognit and funlen keep adapters thin and `cmd/` small. Test files are exempt from both, so integration tests can use real dependencies. The other linters cover the Go guide: exhaustive switches (`exhaustive`, `gochecksumtype`), no package-level state (`gochecknoglobals`, `gochecknoinits`), doc comments on exported names (`revive`), `errors.Is` and `errors.As` (`errorlint`), errors from other modules wrapped where they come in (`wrapcheck`) and `package foo_test` tests (`testpackage`). gosec, modernize and govet's nilness look for bugs and risky code. gosec's G204 and G304, running a command or opening a file named by a variable, are allowed in `cmd/` and adapters, whose job that is. gofumpt and goimports format.

## Tools considered

Decisions on the common Go analysis tools, recorded 28 October 2026. golangci-lint's `standard` set is errcheck, govet, ineffassign, staticcheck and unused.

### In use

| Tool | How |
|---|---|
| Compiler, `go vet`, `internal/` | Toolchain and language. `go vet` runs as golangci-lint's govet |
| staticcheck, U1000 | golangci-lint's `staticcheck` and `unused` |
| errcheck, errorlint, exhaustive, gochecksumtype, depguard, revive | golangci-lint |
| gocyclo and gocognit | Both, deliberately: cyclomatic complexity at most 8, cognitive at most 10, for adapters and the shell |
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
| gosec | golangci-lint. G204 and G304 are excluded in `cmd/` and adapters, and gosec in tests, in the template rather than with `//nolint` |

### Not adopted

| Tool | Why |
|---|---|
| gofmt | Replaced by gofumpt, a strict superset |
| golint | Deprecated; revive replaces it |
| cyclop | Duplicates gocyclo and gocognit |
| revive's function-length rule | Duplicates funlen |
| gomodguard | `verify`'s dependency check, from `go list -deps`, is stricter |
| go-arch-lint, arch-go | depguard, `RoleOf` and the coreboundary, portbranch and configread analysers enforce the architecture more precisely |
| gopls modernize | Covered by golangci-lint's modernize |

### Open

| Tool | Question |
|---|---|
| nilaway | Runs outside golangci-lint and reports false positives, which the no-`//nolint` rule would force code to work around |
| deadcode | Useful only for `command` modules; U1000 may be enough |
| dupl | Whether duplicated code is worth a threshold, given table-driven tests |
| goleak | Needs `goleak.VerifyTestMain` in each package, which `verify` would have to check |
| Semgrep or CodeQL | Whether deeper security analysis is wanted beyond gosec, and as a separate workflow |

## Adding an architecture rule

Rules that need a project's roles or type information across packages are analyzers, built on `golang.org/x/tools/go/analysis`, in `internal/core/analyzers`. To add one:

1. Write a function that returns an `*analysis.Analyzer` for an `analyzers.Project`, and add it to `All`.
2. Add sample code to `internal/core/analyzers/testdata/src`, marking each line that should be reported with a ``// want `regexp` `` comment, and test it with `analysistest.Run`.
3. Describe the rule in `BASELINE.md`, so agents meet it before the analyzer does.

`verify` runs every analyzer over the project, through the adapter in `internal/adapters/gopackages`.

golangci-lint can't extend a shared config, so a project can't add its own rules to `.golangci.yml`. When one needs to, add the option to the baseline.

## Changing a shared setting

1. Change it here: the template in `internal/core/baseline/golangci.yml.tmpl`, the guide in `internal/core/baseline/BASELINE.md`, or the constants in `baseline.go`. To let the core and shell use a third-party package the dependency check rejects, add it to `dependencyAllowlist` with the reason, after reading the code projects call. When bumping golangci-lint, change `GolangciLintVersion` and the version in this repo's `devbox.json` together.
2. `devbox run check`, commit, then `github release . minor`.
3. In go-template and each project:

   ```sh
   devbox run -- go get -tool github.com/ianflanagan1/go-baseline/cmd/go-baseline@latest
   devbox run -- go tool go-baseline write
   devbox run check
   ```

   A project that hasn't done this keeps passing on the old version; one that upgrades without `write` fails `verify`.

## Commands

| Command | What it does |
|---|---|
| `go tool go-baseline write [-kind <kind>]` | Writes `.golangci.yml` for the module in `go.mod` and the module kind (by default, the kind it already records), `BASELINE.md` and `.github/workflows/check.yml` |
| `go tool go-baseline verify` | Lists each way the project differs from the baseline, including I/O dependencies of the core and shell and the analyzers' findings, and fails if there are any |
| `go tool go-baseline test` | Runs the tests with coverage, and fails if a core package is below 80% |
| `go tool go-baseline map` | Lists each package's role, the ports the shell declares and the adapters that implement them |
