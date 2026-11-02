# code-rulebook

[code-rulebook](https://github.com/ianflanagan1/code-rulebook) states the rules every baseline enforces. `TestTheListingFitsThePinnedRulebook` runs `check-listing` from the release `rulebookRelease` pins over `rulebook-coverage.json`, and code-rulebook's `coverage` task reads the listing as committed to build its COVERAGE.md.

| What | Where in the rulebook | Coupling |
|---|---|---|
| The listing's fields, check kinds and coverage levels | `schema/coverage.schema.json` | COUPLED:coverage-listing-shape. = code-rulebook:schema/coverage.schema.json |
| The clause IDs, roles and traits the listing names | `rulebook/` | `check-listing` fails on any it doesn't have, and on any applicable one the listing leaves out |
