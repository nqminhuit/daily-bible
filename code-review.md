# Code Review: `daily-bible`

**Scope:** All Go source files, tests, Makefile, GitHub Actions workflows  
**Reviewer note:** This project is well-structured overall — a focused, single-purpose Go service with good test coverage. The findings below are ordered roughly by severity within each category.

---

## 1. Security

### 1.1 Credentials / Secrets Leakage Risk (CI Workflow)
**File:** `.github/workflows/daily-bible-quote.yml`

The XMPP password is passed via `go-sendxmpp -p "$XMPP_PASS"`, which exposes it in the process list (`ps aux`). Use a credentials file or environment variable that `go-sendxmpp` reads directly instead.

```yaml
# Current (risky)
echo "$QUOTE" | go-sendxmpp -u "$XMPP_JID" -p "$XMPP_PASS" "$recipient"

# Better: use --password-file or check if go-sendxmpp supports XMPP_PASSWORD env
```

### 1.2 Path Traversal in Date Handler
**File:** `internal/api/handlers.go` — `makeDateByPathHandler`

`date` is taken directly from `r.URL.Path` without format validation and passed to a SQL query. Although SQLite parameterization prevents SQL injection, an attacker can probe with arbitrary strings. Add a format check:

```go
if _, err := time.Parse("2006-01-02", date); err != nil {
    http.Error(w, "invalid date format", http.StatusBadRequest)
    return
}
```

### 1.3 No Rate Limiting on Public API
The API has no rate limiting middleware. The `/api/v1/search` and `/api/v1/gospel/` endpoints accept arbitrary input from any origin (CORS `*`). For a public-facing service, add basic rate limiting (e.g., `golang.org/x/time/rate`).

---

## 2. Correctness Bugs

### 2.1 `buildVerseCondition` Single-Verse Path is Broken
**File:** `internal/api/handlers.go` — `buildVerseCondition`

When a verse segment has no dash (single verse), the generated SQL clause is missing the `AND` grouping:

```go
// Produces: "verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL)"
// This is placed inside strings.Join(clauses, " OR "), creating:
// WHERE ... AND (clause1 OR verse = ? AND (verse_suffix = ''))
// Operator precedence: AND binds tighter than OR — this is logically wrong.
clauses = append(clauses, "verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL)")
```

Should be wrapped in parentheses:
```go
clauses = append(clauses, "(verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL))")
```

The same issue doesn't appear in `ref.go`'s `QueryByRef` because it builds conditions differently. This is a divergence that can cause subtle query bugs.

### 2.2 `FetchSitemapAndParse` and `parseSitemap` Are in Different Packages
**File:** `tools/crawler/sitemap.go`

`FetchSitemapAndParse` is exported but `parseSitemap` is unexported. The test file (`sitemap_test.go`) is `package main` (not `package main_test`), so it can access unexported symbols — but `parser_test.go` is `package main_test`. This inconsistency can cause confusion; both should be in `package main` or `package main_test`.

### 2.3 `makeRandomHandler` Has an Off-by-One on Empty DB
**File:** `internal/api/handlers.go`

When `maxRowID = 0`, a zero-argument handler is registered that returns 404. However `NewRouter` computes `maxRowID` once at startup. If rows are inserted after startup (e.g., in a migration scenario), the handler is never updated. This is documented as intentional ("static, immutable table") but isn't enforced — nothing prevents a write after startup.

### 2.4 Race Condition in Crawler Atomics
**File:** `tools/crawler/main.go`

`checked`, `matched`, etc. are package-level `int64` variables used with `atomic`. The `runCrawler` function resets them via `atomic.StoreInt64`. But the tests call `runCrawler` sequentially in the same process — if tests ever run in parallel (e.g., `t.Parallel()`), these resets will race. Move the atomics into a struct passed into `worker`.

### 2.5 `wg.Go` in Tests Uses Go 1.22+ API
**File:** `tools/crawler/main_test.go` — `TestProcessedAndMissingWriters`

```go
var wg sync.WaitGroup
wg.Go(func() { ... })
```

`sync.WaitGroup.Go` was added in Go 1.22. The `go.mod` declares `go 1.25.0` so this compiles, but it's worth noting since the AGENTS.md and CI both say `go-version: '1.25'` — fine, but be aware of the floor.

---

## 3. Error Handling

### 3.1 `resultsWriter` Silently Drops Errors for First 10 Entries
**File:** `tools/crawler/main.go`

```go
count++
if count%10 == 0 {
    if err := w.Flush(); err != nil { ... }
}
```

Write errors on individual `w.WriteString` calls are logged, but the loop continues. A disk-full condition would silently drop all remaining output after logging one error.

### 3.2 `InitDB` Misleading `RowsAffected` Comment
**File:** `internal/db/db.go`

`RowsAffected()` always errors for DDL in SQLite (it returns 0, not an error). The comment says "may not be supported" and logs a message — this fires every time the function is called, polluting logs. Remove the RowsAffected call entirely for DDL.

### 3.3 `makeTodayHandler` Silently Falls Back to UTC
**File:** `internal/api/handlers.go`

```go
loc, err := time.LoadLocation(constants.Timezone)
if err != nil {
    log.Printf("load timezone %q: %v, falling back to UTC", constants.Timezone, err)
    loc = time.UTC
}
```

`"Asia/Ho_Chi_Minh"` is a compile-time constant. If this fails (missing tzdata), the server silently serves wrong dates. This should be a fatal error at startup rather than a silent fallback.

### 3.4 `calendar/main.go` Logs Insert Errors But Continues
**File:** `tools/calendar/main.go`

```go
if _, err := stmt.Exec(...); err != nil {
    log.Printf("insert %s: %v", date, err)
} else {
    inserted++
}
```

A constraint violation (e.g., malformed JSON entry) is silently skipped. If the schema has changed or data is corrupt, the operator gets no clear signal that the import partially failed.

---

## 4. Design / Architecture Issues

### 4.1 `handlers.go` vs `ref.go` — Duplicated Verse-Range Logic
**Files:** `internal/api/handlers.go` (`buildVerseCondition`) and `internal/api/ref.go` (`QueryByRef`)

Both implement reference-to-SQL logic independently. `makeGetGospelHandler` uses `buildVerseCondition` while the CLI and lectionary crawler use `QueryByRef`. The two implementations have different edge-case behaviors (e.g., the single-verse bug in §2.1 only exists in `buildVerseCondition`). `makeGetGospelHandler` should call `QueryByRef` rather than maintaining its own parser.

### 4.2 `makeGetGospelHandler` Re-Parses the URL Manually
**File:** `internal/api/handlers.go`

The handler uses a manual prefix strip + `strings.Fields` + index scan to extract book and chapter. This is fragile and redundant since `ParseRef` in `ref.go` already handles the canonical format. The handler should URL-decode the path and call `QueryByRef` directly.

### 4.3 `maxRowID` Computed at Router Init, Not Refreshed
**File:** `internal/api/router.go` — `NewRouter`

The random handler bakes in `MAX(rowid)` forever. If a re-import is ever done without restarting the server, random will bias toward old rows. A minor issue given the stated immutability assumption, but the assumption is not enforced anywhere.

### 4.4 Crawler Uses Global State, Not Testable in Parallel
**File:** `tools/crawler/main.go`

The six package-level `atomic` counters (`checked`, `matched`, etc.) are global state. This makes the crawler untestable in parallel and its reset behavior in `runCrawler` fragile. These should be encapsulated in a struct.

### 4.5 `tools/lectionarycrawler/main.go` Has No Retry Logic
If a Vatican News page returns a transient error, the key is just skipped with a log line. Given the crawler's purpose (to bootstrap a one-time lookup table), this is probably fine, but a simple retry with backoff would improve reliability.

---

## 5. Testing Gaps

### 5.1 `makeGetGospelHandler` Uses a Different Code Path Than the CLI
As noted in §4.1, the API handler and CLI use different reference parsers. There are no integration tests that verify both paths return identical results for the same reference. A table-driven test comparing `makeGetGospelHandler` and `QueryByRef` for a set of canonical references would catch divergence.

### 5.2 `buildVerseCondition` Has No Direct Unit Tests
**File:** `internal/api/handlers.go`

`buildVerseCondition` and `parseVerseSegments` are untested directly. Their coverage comes only through handler tests with a real DB. Add unit tests for edge cases: single verse with suffix, multi-verse cross-boundary with suffix, invalid input.

### 5.3 `tools/tsv/main_test.go` Changes Working Directory
Tests call `os.Chdir(temp)` without `t.Parallel()` safety. If these tests are ever parallelized, they'll stomp each other's working directory. Use explicit file paths (passed as function arguments) instead of relying on CWD.

### 5.4 No Test for `canonicalizeReference` Edge Cases
**File:** `internal/parser/parser.go`

The reference canonicalization (comma/dash/dot spacing normalization, book+chapter split) has no dedicated unit tests. It's covered incidentally via `TestExtractGospel_ReferenceVariants`, but edge cases like `"1 Cor 1,1-2"` or malformed refs aren't tested.

### 5.5 `TestProcessedAndMissingWriters` Uses `sync.WaitGroup.Go` But No Timeout
**File:** `tools/crawler/main_test.go`

If a writer goroutine panics (e.g., file creation fails), `wg.Wait()` hangs forever. Add a deadline or use `t.Cleanup` with a channel.

---

## 6. Code Quality & Style

### 6.1 `handlers.go` Is Too Long
At ~300 lines, `handlers.go` contains parsing logic (`buildVerseCondition`, `parseVerseSegments`), handler factory functions, and FTS utilities. These should be separated — parsing belongs in `ref.go` (or a dedicated `parse.go`), and health handlers could be in `health.go`.

### 6.2 `FtsPhraseQuery` Is Exported But Only Used Internally + CLI
**File:** `internal/api/handlers.go`

`FtsPhraseQuery` is exported so `cmd/cli/main.go` can call it. Exporting a function solely to share between packages within the same module is a design smell; move it to a shared `internal/query` package or unexport it and duplicate the trivial one-liner.

### 6.3 Magic Number in `makeRandomHandler`
```go
for range 10 {
```
The retry count `10` is unexplained and hardcoded. A named constant with a comment explaining the retry rationale (rowid gaps from deletes) would clarify intent.

### 6.4 `worker` Function Signature Is Too Wide
**File:** `tools/crawler/main.go`

```go
func worker(client *http.Client, jobs <-chan string, results chan<- string,
    done chan<- string, missing chan<- string, failedURLs chan<- string,
    wg *sync.WaitGroup, total int, sleepDuration time.Duration)
```

Nine parameters. Introduce a `WorkerConfig` or `WorkerChannels` struct.

### 6.5 `AGENTS.md` Documents Stale Schema
**File:** `AGENTS.md`

```
## Database
- **Lectionary**: `lectionary(date, book, chapter, verse_start, ...)`
```

The actual schema (`data/migrations/002_lectionary.sql`) is:
```sql
CREATE TABLE lectionary (
    lectionary_key TEXT NOT NULL PRIMARY KEY,
    gospel_ref     TEXT NOT NULL
);
```

The AGENTS.md description is completely wrong. This will mislead contributors and agents.

### 6.6 `go.mod` Specifies `go 1.25.0`
**File:** `go.mod`

Go module files use `go 1.25` (no patch), not `go 1.25.0`. While `go mod` accepts both, the patch suffix is non-standard and may cause tooling inconsistencies.

---

## 7. GitHub Workflows

### 7.1 `actions/checkout@v6` — Non-Existent Version
**Files:** `.github/workflows/ci.yml`, `full.yml`, `daily-bible-quote.yml`

As of this review, `actions/checkout` and `actions/setup-go` are at major version `v4` (with `v4.x` tags). `@v6` will fail at runtime unless the maintainer has pinned a custom fork. Change to `@v4`.

```yaml
# Wrong
- uses: actions/checkout@v6

# Correct (pinned to latest stable)
- uses: actions/checkout@v4
```

### 7.2 `actions/upload-artifact@v7` — Also Non-Existent
**File:** `.github/workflows/full.yml`

Same issue — `upload-artifact` is currently at `v4`. Use `@v4`.

### 7.3 `full.yml` Has No Caching for the Build DB Artifact
**File:** `.github/workflows/full.yml`

The workflow crawls all Vatican News URLs and builds a full DB, then uploads the artifact. If the crawl fails midway, there is no resume logic in CI. The crawler itself supports `processed.txt`/`failed.txt` resumption, but the workflow always starts from scratch. Consider caching `build/processed.txt` between runs.

### 7.4 `daily-bible-quote.yml` — XMPP Recipients Logged in Plain Text
**File:** `.github/workflows/daily-bible-quote.yml`

```yaml
for recipient in $XMPP_RECIPIENTS; do
```

If `XMPP_RECIPIENTS` contains multiple addresses and the loop runs, each is visible in the CI log. Mark the step with `continue-on-error: true` and redirect output, or use a wrapper script.

### 7.5 `daily-bible-quote.yml` — No Error Handling on Fetch Step
```yaml
JSON=$($CMD)
REF=$(echo "$JSON" | jq -r '.ref')
TEXT=$(echo "$JSON" | jq -r '.verses')
```

If `$CMD` exits non-zero (e.g., no reading for today), `JSON` is empty, `jq` silently outputs `null`, and `go-sendxmpp` sends `"null"`. Add `set -e` or explicit error checking:

```yaml
- name: Fetch today's quote
  shell: bash
  run: |
    set -euo pipefail
    ...
```

### 7.6 CI Runs `staticcheck` Before Tests
**File:** `.github/workflows/ci.yml`

The order is: `staticcheck` → `test-with-race-detector` → build. `staticcheck` will fail if the code doesn't compile. This is fine, but `go vet` (faster) is not included, and `staticcheck` installs itself with `go install` every run without caching. Pin the `staticcheck` binary or use `golangci-lint` with caching.

---

## 8. Documentation

### 8.1 `README.md` Shows Wrong Today/Date Response Format
**File:** `README.md`

```json
{
  "verses": [
    {"book": "Ga", "chapter": 11, "verse": 1, "text": "..."}
  ]
}
```

The actual handler (`makeDateHandler`) encodes `verses` as a single concatenated string, not an array of objects:

```go
json.NewEncoder(w).Encode(map[string]any{
    "verses": combined,  // string, not []Gospel
})
```

This will confuse API consumers.

### 8.2 `AGENTS.md` Missing `internal/api/ref.go`
The package structure section omits `internal/api/ref.go` despite it being a significant file containing `ParseRef` and `QueryByRef`.

---

## Summary Table

| # | Severity | Category | File(s) |
|---|----------|----------|---------|
| 1.1 | High | Security | `daily-bible-quote.yml` |
| 1.2 | Medium | Security | `handlers.go` |
| 1.3 | Low | Security | `router.go` |
| 2.1 | High | Correctness | `handlers.go` |
| 2.4 | Medium | Correctness | `main.go` (crawler) |
| 3.1 | Medium | Error Handling | `main.go` (crawler) |
| 3.3 | Medium | Error Handling | `handlers.go` |
| 4.1 | High | Design | `handlers.go`, `ref.go` |
| 4.2 | Medium | Design | `handlers.go` |
| 7.1 | High | CI | All workflows |
| 7.5 | Medium | CI | `daily-bible-quote.yml` |
| 8.1 | Medium | Docs | `README.md` |
| 6.5 | Medium | Docs | `AGENTS.md` |

---

## Quick Wins (Low Effort, High Value)

1. Fix action versions: `@v6` → `@v4`, `upload-artifact@v7` → `@v4` — **will break CI immediately otherwise**
2. Add `set -euo pipefail` to all multi-line `run:` steps in workflows
3. Wrap single-verse clause in `buildVerseCondition` in parentheses
4. Add date format validation in `makeDateByPathHandler`
5. Fix `AGENTS.md` lectionary schema description
6. Fix `README.md` today/date response format
