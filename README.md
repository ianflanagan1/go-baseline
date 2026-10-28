# go-baseline

The settings every Go project shares, and the command that keeps a project on them. It is the Go counterpart of `@ianflanagan1/ts-baseline`; new projects start from [go-template](../go-template).

## Starting a project

From this repo:

```sh
devbox run new github.com/ianflanagan1/<name> ~/projects-ianf/<category>/<name>
```

The directory must be an absolute path, because devbox runs scripts from this repo's root. The script:

1. copies go-template with `gonew`, rewriting the module path in `go.mod` and every import;
2. renames `cmd/app` to `cmd/<name>`, replaces the README and removes the licence, so you choose one;
3. generates `.golangci.yml` for the new module path;
4. runs `devbox run check`;
5. creates a git repo on `main` with one commit.

Publish it with `github publish` from repo-tools.

## What a project gets

| Setting | Where | Checked by |
|---|---|---|
| golangci-lint config, generated from the module path in `go.mod` | `.golangci.yml` | `verify`: must match byte for byte |
| Go pinned to an exact version, golangci-lint pinned to the baseline's version | `devbox.json` packages | `verify` |
| `GOTOOLCHAIN=local` | `devbox.json` env | `verify` |
| `lint` and `check` scripts, and a `test` script | `devbox.json` scripts | `verify` |
| No `src/` or `pkg/` at the root, no `util`, `common` or `helpers` package | directories | `verify` |

`devbox run check` runs `go tool go-baseline verify`, then the linters, then the tests.

## The layout the linters enforce

| Directory | Role | May import |
|---|---|---|
| `internal/core/...` | Functional core: pure functions over values | Other core packages, pure standard library packages, third-party libraries |
| `internal/adapters/...` | Adapters for databases, HTTP, files, the clock and so on | Anything |
| `cmd/<name>` | Composition root: builds adapters and passes them in | Anything |
| Everything else | The shell: reaches outside dependencies only through ports (interfaces) | Anything except adapters and I/O packages |

depguard enforces the import rules, and forbidigo bans the clock functions (`time.Now` and so on) outside `cmd/` and adapters. Test files are exempt from both, so integration tests can use real dependencies. The other linters cover the Go guide: exhaustive switches (`exhaustive`, `gochecksumtype`), no package-level state (`gochecknoglobals`, `gochecknoinits`), doc comments on exported names (`revive`), `errors.Is` and `errors.As` (`errorlint`) and `package foo_test` tests (`testpackage`).

golangci-lint can't extend a shared config, so a project can't add its own rules to `.golangci.yml`. When one needs to, add the option to the baseline.

## Changing a shared setting

1. Change it here: the template in `internal/core/baseline/golangci.yml.tmpl`, or the constants in `baseline.go`. When bumping golangci-lint, change `GolangciLintVersion` and the version in this repo's `devbox.json` together.
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
| `go tool go-baseline write` | Writes `.golangci.yml` for the module in `go.mod` |
| `go tool go-baseline verify` | Lists each way the project differs from the baseline, and fails if there are any |
