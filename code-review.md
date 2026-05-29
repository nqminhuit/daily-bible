# Code Review — daily-bible

**Date:** 2026-05-29
**Scope:** Full codebase (`cmd/`, `internal/`, `tools/`, `data/`, `.github/`)
**Reviewer:** OpenCode (claude-sonnet-4.6)

---

## Summary

The codebase is well-structured for its purpose: a small Go HTTP server and CLI for serving Vietnamese Catholic Gospel readings from a local SQLite database. Package boundaries are sensible, test coverage is reasonable, and the middleware/handler pattern is clean. The main concerns fall into four buckets: broken CI infrastructure, a handful of correctness bugs (including one that produces incorrect API responses), code quality issues that add confusion without adding safety, and a few robustness gaps in the tooling.

---

## 1. Critical — Broken CI

### 1.1 All GitHub Actions workflows use non-existent action versions

**Files:** `.github/workflows/ci.yml`, `daily-bible-quote.yml`, `full.yml`

Every workflow references versions that do not exist:

| Usage | Current | Should be |
|---|---|---|
| `actions/checkout@v6` | all 3 workflows | `@v4` |
| `actions/setup-go@v6` | all 3 workflows | `@v5` |
| `actions/upload-artifact@v7` | `full.yml` | `@v4` |

All CI pipelines will fail at the `uses:` resolution step before running any tests or builds. This means the repo has been operating without working automated CI.

**Fix:** Update the `uses:` pins in all three workflow files.

---

## 2. Bugs

### 2.1 Readiness handler checks the wrong DB path when `DB_PATH` is set

**File:** `internal/api/router.go:14`

```go
mux.HandleFunc("/readiness", makeReadinessHandler(db, constants.DBPath))
```

`NewRouter` always passes the hardcoded `constants.DBPath` (`"resources/bible.db"`) to the readiness handler. The server entrypoint in `cmd/server/main.go` respects the `DB_PATH` env var, but that resolved path is never threaded into `NewRouter`. If the server is started with `DB_PATH=/data/bible.db`, the `/readiness` endpoint checks `resources/bible.db` (which won't exist), returning 503 despite the server being fully operational.

**Fix:** Add a `dbPath string` parameter to `NewRouter` and thread the actual path through.

```go
// cmd/server/main.go
mux := api.NewRouter(db, dbPath)

// internal/api/router.go
func NewRouter(db *sql.DB, dbPath string) http.Handler {
    ...
    mux.HandleFunc("/readiness", makeReadinessHandler(db, dbPath))
    ...
}
```

### 2.2 `today` and `date` endpoints return 200 with empty `verses` field instead of 404

**File:** `internal/api/handlers.go:162-175`

When a `daily_readings` row exists with a `gospel_ref`, but `QueryByRef` returns no verses (e.g., the ref references a book/chapter not in the database), the handler returns HTTP 200 with `"verses": ""` rather than 404. The `gospel` endpoint correctly handles this:

```go
// gospel handler — correct
if len(verses) == 0 {
    http.Error(w, "not found", http.StatusNotFound)
    return
}
```

But the `today`/`date` handler concatenates an empty slice and encodes `"verses": ""` without checking. The same bug exists in `cmd/cli/main.go:outputDateReading`.

**Fix:** Add an empty-verses check before building the response in `makeDateHandler` and `outputDateReading`.

### 2.3 `tools/lectionarycrawler` ignores the `DB_PATH` env var

**File:** `tools/lectionarycrawler/main.go:29`

```go
db, err := dbpkg.Open(constants.DBPath)
```

Every other entry point (`cmd/server`, `cmd/cli`) respects `DB_PATH`. The lectionary crawler hardcodes the path, making it impossible to populate a database at a non-default location without modifying source.

**Fix:** Mirror the pattern from `cmd/server/main.go`:
```go
dbPath := os.Getenv("DB_PATH")
if dbPath == "" {
    dbPath = constants.DBPath
}
db, err := dbpkg.Open(dbPath)
```

### 2.4 `missingGospelRefs` uses `db.Query` without context

**File:** `tools/lectionarycrawler/main.go:128`

```go
rows, err := db.Query(`SELECT DISTINCT dr.lectionary_key ...`)
```

Every query in the API and CLI uses `QueryContext` / `QueryRowContext`. This query is the only one that bypasses context, meaning it cannot be cancelled and will not respect any timeout if one is added later.

**Fix:** Accept a `context.Context` parameter and use `db.QueryContext`.

### 2.5 `FindReadingStartVatican` contains unreachable code

**File:** `internal/parser/parser.go:225-239`

```go
func FindReadingStartVatican(s string) int {
    ls := strings.ToLower(s)
    if i := strings.Index(ls, "tin mừng"); i != -1 {
        return i  // always matches before "tin mừng:" below
    }
    if i := strings.Index(ls, "lời chúa"); i != -1 {
        return i  // always matches before "lời chúa:" below
    }
    if i := strings.Index(ls, "tin mừng:"); i != -1 {  // dead code
        return i
    }
    if i := strings.Index(ls, "lời chúa:"); i != -1 {  // dead code
        return i
    }
    return -1
}
```

`"tin mừng:"` contains `"tin mừng"` as a prefix, so the third condition can never be reached if the second is checked first. Same for `"lời chúa:"`. These branches are dead code and could mask future intent.

**Fix:** Remove the last two `if` blocks. If colon-suffixed variants are needed for disambiguation, restructure to check longer patterns first.

---

## 3. Code Quality

### 3.1 `ParseRef` is called twice for every valid gospel request

**File:** `internal/api/handlers.go:30-39`

```go
if _, _, _, err := ParseRef(ref); err != nil {
    http.Error(w, fmt.Sprintf("invalid reference: %v", err), http.StatusBadRequest)
    return
}
verses, err := QueryByRef(r.Context(), db, ref)  // calls ParseRef internally again
```

`QueryByRef` calls `ParseRef` internally. Every valid request parses the reference string twice. The pre-validation call exists to produce a clean 400 before hitting the database, but the result is discarded. The handler could instead pass the pre-parsed result to a lower-level query function, or simply let `QueryByRef` return the parse error directly (its error would be a parse error before any DB access).

### 3.2 Variable `ref` in search handler is misnamed

**File:** `internal/api/handlers.go:68`

```go
var ref string
if err := rows.Scan(&ref); err != nil {
```

The column being scanned is `text FROM verses_fts`. The variable should be named `text`. This makes the code misleading — `results = append(results, ref)` looks like a reference is being collected, not verse text.

### 3.3 `verse_suffix IS NULL` checks are dead code

**File:** `internal/api/ref.go:66, 77, 85`

The schema defines `verse_suffix TEXT NOT NULL DEFAULT ''`. The column can never be NULL. The generated SQL contains conditions like:

```sql
(verse_suffix = '' OR verse_suffix IS NULL)
```

The `IS NULL` arm is dead code. It adds noise to generated queries without effect.

**Fix:** Remove the `IS NULL` branches in `QueryByRef` and simplify to `verse_suffix = ''`.

### 3.4 `db == nil` check in `makeSearchHandler` is inconsistently applied

**File:** `internal/api/handlers.go:57`

```go
if db == nil {
    http.Error(w, "internal server error", http.StatusInternalServerError)
    return
}
```

No other handler has this guard. `NewRouter` never passes a nil DB, so this check will never trigger. It creates a false impression that nil DB safety is a general concern handled throughout the codebase.

**Fix:** Either add the guard consistently to all handlers, or remove it from `makeSearchHandler`.

### 3.5 `setupTestDB` is duplicated across test files in the same package

**Files:** `internal/api/handlers_test.go:14`, `internal/api/ref_test.go:14`

`setupTestDB` and `setupTestDBWithLectionary` are functionally identical: both open `:memory:` and apply all migration files. They live in the same package (`package api`), so one can be removed. The dual presence suggests they evolved independently.

**Fix:** Keep one helper (the more descriptive `setupTestDBWithLectionary` name, or a neutral `newTestDB`) and delete the other.

### 3.6 `TestGetGospel` is a redundant subset of `TestGetGospel_ValidAndErrors`

**File:** `internal/api/handlers_test.go:42`

`TestGetGospel` inserts a verse and asserts a 200. The same scenario is covered more thoroughly in `TestGetGospel_ValidAndErrors`. The first test adds no coverage.

### 3.7 `TestNewRouter_HealthHandlers` leaks files in the source tree

**File:** `internal/api/router_test.go:46-58`

```go
resourcesDir := "resources"
if err := os.MkdirAll(resourcesDir, 0o755); err != nil { ... }
dbFile := filepath.Join(resourcesDir, "bible.db")
if err := os.WriteFile(dbFile, []byte("ready"), 0o644); err != nil { ... }
defer os.Remove(dbFile)
```

`os.MkdirAll` creates `internal/api/resources/` in the source tree. `defer os.Remove` removes the file but leaves the directory. If the test is interrupted before the deferred cleanup, the file lingers. This happens because `makeReadinessHandler` takes a hardcoded path derived from `constants.DBPath`.

The root cause is that `NewRouter` hardcodes `constants.DBPath` for the readiness check (see bug 2.1). Once that is fixed to accept a configurable path, the test can write its temp file into `t.TempDir()` and pass that path to `NewRouter`.

### 3.8 JSON encoding errors are silently discarded everywhere

All handlers and CLI commands use `json.NewEncoder(w).Encode(...)` without checking the return error. For a `http.ResponseWriter`, encoding errors typically indicate the connection was dropped mid-response, which is not actionable. But for CLI stdout, a write failure (e.g., broken pipe) could silently produce no output. This is a minor quality issue common in Go HTTP code, but worth a note.

### 3.9 `utcLoc` is an unnecessary alias for `time.UTC`

**File:** `internal/lectionary/types.go:18`

```go
var utcLoc = time.UTC
```

`time.UTC` is already a package-level variable in the standard library. This alias adds no value.

### 3.10 `tx.Rollback()` error discarded after failed `Commit()`

**File:** `tools/lectionarycrawler/main.go:120`

```go
if err := tx.Commit(); err != nil {
    tx.Rollback()  // error ignored
    return fmt.Errorf("commit: %w", err)
}
```

The idiomatic Go pattern is `defer tx.Rollback()` before the commit, which is a no-op after a successful commit and handles the error case automatically.

### 3.11 CORS middleware sets `Allow-Methods` and `Allow-Headers` on every response

**File:** `internal/api/router.go:40-43`

```go
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
```

`Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` are only meaningful in preflight (OPTIONS) responses. Setting them on all responses is harmless but adds noise to every response header.

### 3.12 `constants` package conflates server and crawler concerns

**File:** `internal/constants/constants.go`

Server constants (`DBPath`, `ServerAddr`, `Timezone`) are in the same package as crawler output paths (`OutFilename`, `LinkFile`, `ProcessedFile`, etc.) and crawler configuration (`Workers`, `VaticanPrefix`). The server imports this package but doesn't use the crawler constants; the crawler imports it and uses everything. As the project grows this creates unnecessary coupling.

### 3.13 `GenerateYear` returns `DayInfo` with `WeekOfSeason = 0`

**File:** `internal/lectionary/generators.go:9-38`

`GenerateYear` is exported but returns `DayInfo` with `WeekOfSeason = 0` for all entries; week numbers are only populated by `applyWeekNumbers`, which is unexported. A caller using `GenerateYear` directly gets incomplete `DayInfo`. `LectionaryKey` is also unpopulated until `lectionaryPass` runs. Either `GenerateYear` should be unexported, or its documentation should warn that the result is incomplete without the two subsequent passes.

---

## 4. Robustness

### 4.1 Writer goroutines in `tools/crawler` use `panic` on file open failure

**File:** `tools/crawler/main.go:254-258, 267-271, 279-283`

```go
func processedWriter(filename string, ch <-chan string) {
    f, err := os.OpenFile(...)
    if err != nil {
        panic(err)  // crashes the whole process from a goroutine
    }
    ...
}
```

Using `panic` in a goroutine propagates and crashes the whole process. While the effect (crashing) may be acceptable for a CLI tool, `log.Fatal` would be more intentional and consistent with the rest of the codebase.

### 4.2 `writeLinksToFile` panics on file close error

**File:** `tools/crawler/main.go:332-336`

```go
defer func() {
    if err := out.Close(); err != nil {
        panic(fmt.Errorf("failed to close file: %w", err))
    }
}()
```

A file close failure on write (e.g., disk full) is worth surfacing, but panicking is aggressive. Returning the error (or `log.Fatal`) would be more appropriate and allow the caller to handle it.

### 4.3 `resultsWriter` silently drops crawl data on write failure

**File:** `tools/crawler/main.go:302-312`

```go
if writeFailed {
    continue  // drains channel but discards all remaining results
}
```

If a write error occurs mid-crawl, all subsequent successful crawl results are silently discarded while their URLs are still recorded as `done`. Re-running the crawler won't re-process them because they're marked processed. The combination of silent data loss + marking URLs as processed means data may be permanently lost without any warning beyond the logged error message.

**Fix:** If a write error occurs, stop marking URLs as processed (or write to a separate error file) so a re-run can recover the lost results.

### 4.4 No upper bound on verse range size in `QueryByRef`

**File:** `internal/api/ref.go:58-86`

For a range with suffixes spanning many verses (e.g., a malformed ref like `Ga 1,1a-100b`), the loop `for v := r.Start; v <= r.End; v++` generates one SQL clause per verse. The HTTP API parses refs from user-supplied URL path segments, so a crafted URL like `/api/v1/gospel/Ga%201,1a-1000b` would generate 1,000 SQL OR clauses. In practice, the attacker can only cause extra DB work (not injection), but the unbounded behavior is worth noting.

**Fix:** Add a maximum span check (e.g., reject any range where `End - Start > 200`).

---

## 5. Observations (No Action Required)

These are noted for completeness but are not defects.

- **`internal/api/handlers.go:makeTodayHandler`** calls `log.Fatalf` at construction time if the timezone fails to load. This is deliberate fail-fast behavior, but it means any error in `NewRouter` will crash the process before the server starts — acceptable but non-obvious to new contributors.

- **`internal/lectionary/calendar.go:weekdayCycle`** uses the calendar year (not the liturgical year) to determine cycle I/II. Odd years = cycle I, even years = cycle II. This is a deliberate design choice but different from how Sunday cycles are computed.

- **`internal/parser/parser.go:inferBookFromHeader`** hardcodes Vietnamese Gospel titles. This is correct and intentional (the upstream source is Vietnamese Catholic content).

- **`tools/tsv/main.go`** performs its own duplicate-key detection in memory (`seen` map). This is fine for the current data size but would need re-evaluation for very large inputs.

- **`internal/lectionary/generators.go:applyWeekNumbers`** — the backward pass from Advent that assigns post-Lent ordinary week numbers (week 34 counting down) appears correct: it terminates when it reaches a non-ordinary season (Easter/Pentecost), long before `ordWeek` would reach 0.

---

## Issue Index

| # | Severity | File | Description |
|---|---|---|---|
| 1.1 | Critical | `.github/workflows/*.yml` | All action versions non-existent (`@v6`, `@v7`) |
| 2.1 | Bug | `internal/api/router.go:14` | Readiness handler checks wrong DB path when `DB_PATH` is overridden |
| 2.2 | Bug | `internal/api/handlers.go:162` | `today`/`date` return 200 with empty verses instead of 404 |
| 2.3 | Bug | `tools/lectionarycrawler/main.go:29` | Lectionary crawler ignores `DB_PATH` env var |
| 2.4 | Bug | `tools/lectionarycrawler/main.go:128` | `missingGospelRefs` uses `db.Query` without context |
| 2.5 | Bug | `internal/parser/parser.go:225` | Last two branches in `FindReadingStartVatican` are dead/unreachable |
| 3.1 | Quality | `internal/api/handlers.go:30` | `ParseRef` called twice per valid gospel request |
| 3.2 | Quality | `internal/api/handlers.go:68` | Variable `ref` should be `text` in search handler |
| 3.3 | Quality | `internal/api/ref.go:66` | `verse_suffix IS NULL` is dead code (column is NOT NULL) |
| 3.4 | Quality | `internal/api/handlers.go:57` | `db == nil` guard only in `makeSearchHandler`, not other handlers |
| 3.5 | Quality | `internal/api/*_test.go` | `setupTestDB` duplicated across test files in same package |
| 3.6 | Quality | `internal/api/handlers_test.go:42` | `TestGetGospel` is a redundant subset of `TestGetGospel_ValidAndErrors` |
| 3.7 | Quality | `internal/api/router_test.go:46` | Test creates files in source tree instead of `t.TempDir()` |
| 3.8 | Quality | All handlers and CLI | JSON encoding errors silently discarded |
| 3.9 | Quality | `internal/lectionary/types.go:18` | `utcLoc` is unnecessary alias for `time.UTC` |
| 3.10 | Quality | `tools/lectionarycrawler/main.go:120` | `tx.Rollback()` error ignored after `Commit()` failure |
| 3.11 | Quality | `internal/api/router.go:42` | CORS `Allow-Methods`/`Allow-Headers` set on every response, not just preflight |
| 3.12 | Quality | `internal/constants/constants.go` | Server and crawler constants conflated in one package |
| 3.13 | Quality | `internal/lectionary/generators.go:9` | `GenerateYear` returns incomplete `DayInfo` (week numbers = 0) |
| 4.1 | Robustness | `tools/crawler/main.go:254` | Writer goroutines panic on file open failure |
| 4.2 | Robustness | `tools/crawler/main.go:332` | `writeLinksToFile` panics on file close error |
| 4.3 | Robustness | `tools/crawler/main.go:302` | Silent data loss in `resultsWriter` on write failure |
| 4.4 | Robustness | `internal/api/ref.go:58` | No upper bound on verse range span in `QueryByRef` |
