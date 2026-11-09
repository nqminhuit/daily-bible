# Handoff — Code review remediation complete

## Current state

All five original implementation groups from `PLAN.md` are complete, pushed
through `7d4b8a5`, and passing remote CI. The user subsequently approved optional
issue 3.12, separating runtime and crawler constants. That follow-up is complete
and locally validated, delivered in a separate commit, and not yet pushed.
Remote CI still needs to validate the constants split when it is pushed.

- Repository: `/opt/minh/test/daily-bible`
- Detailed tracker and validation history: `PLAN.md`
- Original review: `code-review.md` (contains outdated findings; see corrections below)
- Project rules: `AGENTS.md`
- No review implementation work remains. The constants split is no longer deferred.

## Delivered work

| Commit | Work |
|---|---|
| `dd125be` | Recorded the corrected review findings and implementation plan. |
| `fa476b9` | Group 1: configurable readiness DB path, temporary readiness files, API 404 on readings without matching verses, crawler `DB_PATH`, regression tests. |
| `e2300dc` | Group 2: serialized crawler persistence, payload flush before processed entry, cancellation on persistence failure, startup/shutdown cleanup, checked close errors, honored sleep duration, failure-injection tests. |
| `7d44b91` | Group 3: parse references once, strict numeric/range validation, input complexity limits, bounded suffix SQL, safe cross-chapter expansion, fuzz tests and documentation. |
| `9d24188` | Group 4: signal/context cancellation throughout the lectionary crawler, reliable transaction rollback, checked CLI JSON writes, logged API output failures, search row-error handling, regression tests. |
| `7d4b8a5` | Group 5: consolidated API test DB setup, removed redundant test/dead parser branches/isolated search nil guard, renamed search text variable, removed UTC alias, documented `GenerateYear`, restricted CORS preflight headers, new CORS and marker tests, final tracking updates. |
| Constants follow-up (`refactor: separate crawler and runtime constants`) | User-approved issue 3.12: moved crawler/pipeline defaults into `internal/crawlerconfig`, retained runtime defaults in `internal/constants`, updated crawler/TSV imports and package documentation, checked dependency boundaries. |

Use `git log -1` to obtain the constants follow-up commit hash after delivery.

## Constants split follow-up

- `internal/constants/constants.go` retains only `DBPath`, `ServerAddr`, and
  `Timezone`; existing server, CLI, date utilities, and lectionary crawler callers
  remain unchanged.
- `internal/crawlerconfig/constants.go` owns crawler settings, Vatican URLs, and
  pipeline artifact paths shared by `tools/crawler` and `tools/tsv`.
- All constant names and values are preserved; no behavioral change is intended.
- Updated imports in `tools/crawler/main.go`, `tools/crawler/sitemap_test.go`, and
  `tools/tsv/main.go`, and package documentation in `README.md` and `AGENTS.md`.
- Full uncached FTS5 tests/race suite, all-package build, Staticcheck, actionlint,
  and diff checks passed again after the split.
- `go list -tags "fts5" -deps` confirms runtime entrypoints do not depend on
  crawlerconfig, and crawler/TSV do not depend on runtime constants.
- The follow-up was not pushed; no workflow, database, generated artifact, or
  module dependency changes were made.

## Group 5 file summary

- `internal/api/handlers.go`: search variable renamed and isolated nil guard removed;
  missing/oversized query validation remains before database access.
- `internal/api/handlers_test.go`: one shared helper using `db.Open(":memory:")`,
  `ApplyMigrations`, production-equivalent single-connection pooling, and `t.Cleanup`.
  Removed redundant success test; the scenario remains covered more thoroughly in
  `TestGetGospel_ValidAndErrors`.
- `internal/api/ref_test.go`: removed duplicate database helper and updated callers.
- `internal/api/router.go`: preflight-only allow-methods/allow-headers now on OPTIONS
  only; wildcard allow-origin retained on normal responses.
- `internal/api/cors_test.go`: covers preflight/GET headers, downstream invocation,
  and empty 204 preflight response.
- `internal/lectionary/{calendar.go,calendar_test.go,generators.go,types.go}`:
  use `time.UTC` directly; document `GenerateYear` as an intermediate generation step.
  gofmt also aligned existing fields in touched files.
- `internal/parser/parser.go`: removed unreachable colon-specific marker branches.
- `internal/parser/reading_start_test.go`: marker, colon, uppercase, offset,
  missing-marker, and retained Gospel-priority coverage.

## Final validation

All passed on local **Go 1.26.1 linux/amd64**:

```sh
go test -tags "fts5" ./...
go test -tags "fts5" -race ./...
go test -tags "fts5" -count=1 ./...
go test -tags "fts5" -race -count=1 ./...
go build -tags "fts5" ./...
go test -tags "fts5" ./internal/api ./internal/parser ./internal/lectionary \
  -run 'TestCORSMiddleware|TestFindReadingStartVatican|TestGenerateYear|TestGetGospel_ValidAndErrors'
staticcheck -tags "fts5" ./...
actionlint
git diff --check
git diff --cached --check
```

Staticcheck **2026.2.2 (0.8.2)** and actionlint **1.7.12** were installed into
`/tmp/daily-bible-review-tools.G4kXrj`; they are not globally on PATH and that temporary
directory may later disappear. No project dependencies were changed for tooling.

Earlier validation also included 10 repeated crawler race runs, 10 repeated API/
lectionary-crawler race runs, and 803,270 parser fuzz executions without failure.
SQLite dependency C compiler const-qualifier warnings were observed earlier but
were not failures.

Final diff inspection found no changes to `resources/`, generated `build/` files,
workflows, `go.mod`, or `go.sum`.

## Workflow/remote CI verification

- Authenticated `gh` access showed the latest five remote CI runs successful.
- Inspected [run 32461592865](https://github.com/nqminhuit/daily-bible/actions/runs/32461592865)
  for commit `e83c351ab93bc33e335700b4499424073d65c0b2`: checkout, Go setup,
  Staticcheck, tests, and build all succeeded.
- Subsequently pushed groups 1–5 through `7d4b8a5c29856dd41263c675d20990f4f525ec7a`.
  [CI run 42572360310](https://github.com/nqminhuit/daily-bible/actions/runs/42572360310)
  passed Staticcheck, tests with race detection, and the server build for that
  exact commit. This does not validate the later, unpushed constants split.
- CI targets Go 1.25; local final checks used Go 1.26.1.
- No workflow versions were changed and no manual workflow dispatch was performed.

## Corrections and retained limitations

- The review's nonexistent-action-version claim was false: checkout v6,
  setup-go v6, and upload-artifact v7 tags exist. Do not downgrade them based on
  the review. Current workflows also pass actionlint.
- The CLI already rejected zero matching verses before group 1; that response
  defect was API-only. CLI regression coverage was added.
- Reference limits are 1,024 bytes and 32 ranges after cross-chapter expansion,
  not a blanket verse-span limit. End-of-chapter uses `math.MaxInt`; forward
  cross-chapter ranges include intervening chapters within the range budget.
- Crawler payload/state files are ordered but not transactional or fsync-durable.
  Interrupted runs can leave partial payloads or duplicates; there is no
  exactly-once/crash-durability guarantee.
- Ordinary remote-fetch failures in the lectionary crawler still log/continue;
  database update errors fail the run and cancellation stops subsequent jobs.
- Review issue **3.12**, originally deferred as optional organization work, was
  subsequently approved by the user and completed in the constants split follow-up.

## Future-session rules

Always include `-tags "fts5"` with `go test`, `go build`, and `go run`. Preserve
single-connection SQLite tests, temporary crawler state paths, and zero-delay
worker tests. Do not modify the production database for validation or commit
`build/` artifacts. Consult `PLAN.md` before reopening a completed review finding.
