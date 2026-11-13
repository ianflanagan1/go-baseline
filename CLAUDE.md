# go-baseline

See README.md. `devbox run -- task check` runs everything, and must pass before finishing a change.

@BASELINE.md

- Go, golangci-lint and Task come from the project's devbox. Run one-off commands as `devbox run -- go …` rather than installing Go.
- After changing `internal/core/rulebook/rulebook.go`, run `devbox run -- task rulebook-coverage`, then `check`.
- The modules under `testdata` break the rules on purpose, for the conformance and analyzer tests.
- `docs/analyzers-and-verify-steps-to-build.md` lists every analyzer and `verify` step the baseline still needs. When a clause needs one the baseline doesn't have, or a requirement for a planned one arrives or changes, add or change its entry in the same change, in the form the file gives, and don't build the check in that change. Build one check to a session, test-first, and delete its entry in the change that adds it. The tests in `internal/core/rulebook/checkstobuild_test.go` fail on an entry that is incomplete, that names a clause the listing lacks or covers in full, or that names a check that exists.
