# Code Review Fix Plan

## Status and working rules

- Source review: `code-review.md` (dated 2026-05-29).
- Groups 1–3 complete: response/configuration correctness, crawler persistence/shutdown safety, and reference parsing/query hardening. Remaining groups are tracked below.
- Relevant current source files and upstream GitHub Action tags were inspected. Tests have not yet been run for this plan.
- Check off tasks as they are completed. Record validation results and deviations in the progress log below so another agent/session can resume.
- Always use `-tags "fts5"` for `go test`, `go build`, and `go run`, per `AGENTS.md`.
- Keep changes in focused commits. Do not commit generated `build/` artifacts or modify the production database as part of testing.

## Corrections to the original review

- **1.1 is not a confirmed CI defect.** Upstream tags `actions/checkout@v6`, `actions/setup-go@v6`, and `actions/upload-artifact@v7` all exist (verified with `git ls-remote`). Do not downgrade them based on the review. The current workflows are `ci.yml`, `full.yml`, `heal-readings.yml`, and `update-lectionary.yml`; `daily-bible-quote.yml` is absent, and `full.yml` no longer references `upload-artifact`. Tag existence alone does not prove CI passes; validate workflow syntax and actual runs separately.
- **2.2 is already fixed in the CLI.** `cmd/cli/main.go:outputDateReading` already rejects zero matching verses. The API still needs the equivalent check.
- **4.4 needs a different implementation than a blanket span limit.** Cross-chapter parsing currently uses `999` as an end-of-chapter sentinel. A blanket 200-verse cap would reject valid references. Prefer bounded-size SQL predicates and input complexity limits.
- **4.1 should not be fixed by replacing panic with `log.Fatal`.** Both abruptly terminate the process. Propagate writer errors to the crawler coordinator and shut down cleanly.
- Review line numbers and some observations are stale; use current code as the implementation baseline.

## 1. Response correctness and database configuration

**Review issues:** 2.1, 2.2, 2.3, 3.7

**Primary files:** `internal/api/router.go`, `internal/api/handlers.go`, `cmd/server/main.go`, `tools/lectionarycrawler/main.go`, and related tests.

- [x] Change `NewRouter` to accept the resolved database path and pass it to the readiness handler.
- [x] Update the server and every router test/caller to use the new signature.
- [x] Move readiness test files from the source tree into `t.TempDir()`.
- [x] Return HTTP 404 from date/today handlers when the stored reference resolves to zero verses.
- [x] Respect `DB_PATH` in the lectionary crawler, with `constants.DBPath` as the fallback.
- [x] Preserve the CLI's existing empty-verses rejection.
- [x] Add regression coverage for overridden readiness paths and missing database files.
- [x] Add date/today tests with a stored reference but no matching verses, and verify the CLI equivalent.
- [x] Test crawler database path resolution without touching the production database or fetching live pages.

**Acceptance:** readiness checks the configured database; missing verse data is not reported as a successful empty reading; crawler path selection matches other entry points.

## 2. Crawler persistence and shutdown safety

**Review issues:** 4.1, 4.2, 4.3

**Priority:** highest data-integrity risk.

**Primary files:** `tools/crawler/main.go` and its tests.

- [x] Carry URL and payload together so persistence results can be associated with the correct URL.
- [x] Mark a URL processed only after its payload is successfully written and flushed, including missing-verse payloads.
- [x] Propagate output open/write/flush/close errors to `runCrawler` and return failure instead of silently discarding subsequent results.
- [x] Coordinate cancellation, channel closure, and worker/writer completion to prevent deadlocks on writer failure.
- [x] Open output files before starting workers and ensure all early-return paths release resources and stop goroutines.
- [x] Make `writeLinksToFile` return close failures rather than panic, retaining any earlier write/flush error.
- [x] Add injectable failing writers or equivalent deterministic test seams.
- [x] Test write and flush failure, output-open failure, clean shutdown, and successful persistence-before-processed ordering.
- [x] Verify failed/unpersisted payloads remain retryable and are not put into a state file that causes them to be skipped on rerun.
- [x] Document recovery guarantees: separate payload/state files may permit duplicates after interruption; they must not skip unwritten payloads. Do not claim crash-durable or exactly-once persistence without implementing it.

**Related observations to address in this refactor:** writer goroutines currently start before sitemap setup can fail; `runCrawler` also ignores its supplied `sleepDuration` when configuring workers. Preserve zero-delay tests and use the supplied duration.

**Acceptance:** persistence errors produce an unsuccessful run, do not mark lost payloads processed, and do not leave blocked goroutines.

## 3. Reference parsing and query hardening

**Review issues:** 3.1, 3.3, 4.4

**Primary files:** `internal/api/ref.go`, `internal/api/handlers.go`, and reference/handler tests.

- [x] Parse a gospel request once and pass the parsed representation to an internal query helper; retain `QueryByRef` for existing callers.
- [x] Preserve HTTP 400 for malformed user references and HTTP 500 for database failures.
- [x] Replace per-verse SQL expansion with bounded-size start/interior/end predicates while preserving suffix semantics.
- [x] Define and document reference-length and segment-count limits to bound query construction work.
- [x] Check numeric conversion errors, reject reversed ranges, and require complete segment matches rather than accepting arbitrary matching substrings.
- [x] Preserve supported cross-chapter behavior and account for the current end-of-chapter sentinel.
- [x] Remove redundant `verse_suffix IS NULL` conditions after confirming the NOT NULL schema contract.
- [x] Test suffix boundaries, dot-separated segments, cross-chapter references, oversized inputs, integer overflow, reversed ranges, and malformed trailing content.

**Acceptance:** supported references retain their results; invalid inputs fail before database work; SQL size is not proportional to verse span and is bounded by validated input complexity.

## 4. Cancellation and error reporting

**Review issues:** 2.4, 3.8, 3.10

**Primary files:** `tools/lectionarycrawler/main.go`, `cmd/cli/main.go`, `internal/api/handlers.go`, and related tests.

- [ ] Thread a cancellable context through lectionary crawler queries, transactions, updates, and HTTP requests, not only `missingGospelRefs`.
- [ ] Make retry/throttle waits respond to cancellation and stop the crawl on cancellation rather than continuing through jobs.
- [ ] Defer transaction rollback immediately after successful transaction creation; preserve the primary operation/commit error.
- [ ] Report CLI JSON encoding errors and exit unsuccessfully.
- [ ] Log HTTP JSON encoding errors without attempting to send a second response after output has begun.
- [ ] Check `rows.Err()` in the API search handler before encoding a successful response (additional issue found during inspection).
- [ ] Test cancelled operations, transaction failure/cleanup, and failing output writers.

**Acceptance:** cancellation reaches outstanding work, cleanup is reliable, and output failures are not silently reported as success.

## 5. Low-risk cleanup

**Review issues:** 2.5, 3.2, 3.4, 3.5, 3.6, 3.9, 3.11, 3.13

- [ ] Remove unreachable colon-specific branches from `FindReadingStartVatican`; retain marker-matching tests.
- [ ] Rename the search handler's scanned `ref` variable to `text`.
- [ ] Remove the isolated search nil-DB guard; preserve input validation before database access. Production router construction requires a valid database.
- [ ] Consolidate API test database helpers, use `t.Cleanup`, and enforce the single-connection pool required for in-memory SQLite tests.
- [ ] Remove the redundant `TestGetGospel` only after confirming its assertions remain covered.
- [ ] Replace `utcLoc` with `time.UTC`, including test references.
- [ ] Set `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` only for OPTIONS responses; keep `Access-Control-Allow-Origin` on normal responses.
- [ ] Test normal and preflight CORS responses.
- [ ] Document `GenerateYear` as an intermediate calendar-generation step with unpopulated week/key fields; do not change its semantics unnecessarily.

**Deferred:** issue 3.12 (splitting server/crawler constants) is optional organizational work, not a correctness fix. Revisit only if it provides a concrete benefit.

## Validation and delivery

- [x] Establish a baseline test result before implementing fixes; record pre-existing failures separately.
- [ ] Run focused package/regression tests after each implementation group.
- [ ] Run the full test suite with FTS5.
- [ ] Run the race detector, especially for crawler shutdown changes.
- [ ] Build all packages and run static analysis with FTS5 enabled.
- [ ] Validate current workflow syntax and inspect actual CI results; do not infer CI failure from the action major versions.
- [ ] Review the final diff for unrelated changes, generated files, and accidental database modifications.
- [ ] Update this plan with completed work, commands/results, remaining limitations, and any follow-up issues.

```sh
go test -tags "fts5" ./...
go test -tags "fts5" -race ./...
go build -tags "fts5" ./...
staticcheck -tags "fts5" ./...
```

**Suggested commit order:**

1. API/configuration correctness and readiness test isolation.
2. Crawler persistence and shutdown safety.
3. Reference parsing/query hardening.
4. Cancellation and error reporting.
5. Low-risk cleanup.

## Progress log

- Initial plan: checked the review against current code, verified upstream action tags, identified stale findings, and recorded implementation groups and acceptance criteria. Plan committed as `dd125be`.
- Baseline: `go test -tags "fts5" ./...` passed. The SQLite dependency emitted C compiler const-qualifier warnings, but no tests failed.
- First fix (2.1, 3.7): `NewRouter` now accepts the resolved database path from the server. Router tests use `t.TempDir()` and verify readiness returns 200 for the custom path and 503 after that file is removed; liveness remains 200. Focused API/server tests, the full FTS5 test suite, and `git diff --check` passed. These changes are included in the group 1 implementation commit.
- Group 1 completed: date/today API handlers now return 404 for a stored reference without matching verses; CLI subprocess tests confirm nonzero exit, no stdout, and a not-found diagnostic for both commands. The lectionary crawler resolves `DB_PATH` via a tested helper with the existing default fallback. Validation passed: `go test -tags "fts5" ./...`, `go test -tags "fts5" -race ./internal/api ./cmd/cli ./tools/lectionarycrawler`, `go build -tags "fts5" ./...`, and `git diff --check`. Staticcheck and the full-project race run remain for later validation. Group 1 committed as `fa476b9`.
- Group 2 completed: replaced independent output/state writer goroutines with a mutex-serialized `crawlSink` receiving URL/payload records. Payloads are flushed before processed entries; the first persistence error is retained and cancels all worker HTTP requests, job enqueueing, and throttle waits. Workers are joined before outputs close. The supplied `sleepDuration` is now honored. Setup failures close outputs without starting workers, and links-file close errors are returned alongside earlier errors.
- Group 2 regression coverage: deterministic write/flush/state/close failures, payload-before-processed ordering, sticky errors and retryable state, output-open failure, sitemap/links setup cleanup, in-flight request cancellation, and coordinator shutdown with 1,000 queued URLs for both normal and missing-verse payloads. Reworked existing writer tests to wait for synchronous persistence rather than arbitrary sleeps. `TestProcessedAndMissingWriters` still uses `t.TempDir()`.
- Recovery limitation: flushing is not `fsync` and separate files are not transactional. Interrupted runs can leave partial payloads or duplicate records on retry; no crash-durability or exactly-once guarantee is claimed. Write/flush failures stop the run before the affected URL is marked processed; close failures are surfaced to the caller.
- Group 2 validation passed: full FTS5 tests, full FTS5 race suite, all-package FTS5 build, `go test -tags "fts5" -race -count=10 ./tools/crawler`, and `git diff --check`. Staticcheck is not installed in this environment and was not run. Group 2 committed as `e2300dc`.
- Group 3 completed: gospel handlers parse once and query validated ranges through an internal helper; `QueryByRef` remains available. Reference validation now rejects incomplete matches, empty segments, nonpositive/overflowing numbers, reversed ranges, and non-forward cross-chapter ranges. Limits are 1,024 raw reference bytes (decoded for HTTP) and 32 ranges after expansion, documented in `README.md`.
- Group 3 query changes: replaced per-verse suffix expansion with at most three predicates per range and removed NULL suffix branches under the NOT NULL schema contract. End-of-chapter ranges now use `math.MaxInt` rather than 999, with no endpoint arithmetic in SQL generation. Forward ranges include intervening chapters instead of silently omitting them; expansion is checked against the range budget before looping.
- Group 3 regression coverage: malformed/oversized references are rejected before database access and return HTTP 400; numeric overflow, exact input/range limits, suffix boundary combinations, overlapping/dot ranges, cross-chapter suffixes and intermediate chapters, verses above the old 999 cutoff, and MaxInt spans. Existing DB-error/404 and CLI tests remain passing. A 10-second parser fuzz run with two workers completed 803,270 executions without failure.
- Group 3 validation passed: full FTS5 tests, full FTS5 race suite, all-package FTS5 build, focused API/CLI tests, parser fuzzing, and `git diff --check`. Staticcheck remains unavailable. Next: group 4, cancellation and error reporting.
