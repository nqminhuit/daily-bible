# AGENTS.md — daily-bible

## Critical Build Rule

**Always use `-tags "fts5"`** for `go build`, `go test`, `go run`. The Makefile sets `GOFLAGS=-tags "fts5"`. Without this tag, the FTS5 virtual table won't compile and tests will fail.

```shell
go test -tags "fts5" ./...
go build -tags "fts5" ./...
```

## Commands

| Task | Command |
|------|---------|
| Available targets | `make help` |
| Run tests | `make test` |
| Tests with race detector | `make test-with-race-detector` |
| Single test | `go test -tags "fts5" ./internal/api -run TestName` |
| Build server binary | `make build` → `build/daily-bible-server` |
| Build CLI binary | `make build-cli` → `build/daily-bible` |
| Dev server | `make dev` (listens on `:8090`, override with `PORT=:8080`) |
| CI lint | `staticcheck ./...` |

## Data Pipeline

```
Vatican News sitemap → tools/crawler → build/gospels.txt → tools/tsv → build/gospels.tsv → make import-db → build/bible.db
```

- Crawler state lives in `build/` (`processed.txt`, `failed.txt`, `missing_verse_number.txt`)
- To retry failed URLs after parser changes: delete `build/failed.txt` before re-running crawler
- `build/` is gitignored — never commit generated artifacts

## Database

- **Runtime path**: `build/bible.db`, overridable via `DB_PATH` env var
- **Schema**: `verses(book, chapter, verse, verse_suffix, text)` — composite PK includes `verse_suffix`
- **Lectionary**: `lectionary(lectionary_key TEXT PRIMARY KEY, gospel_ref TEXT NOT NULL)` maps a liturgical key to a Gospel reference
- **FTS**: `verses_fts` uses `unicode61 remove_diacritics 2` with prefix indexing, synced by triggers
- **Connection pool**: `SetMaxOpenConns(1)` — SQLite is single-connection

## CLI

- `build/daily-bible today` — same as `GET /api/v1/today`
- `build/daily-bible gospel <ref>` — same as `GET /api/v1/gospel/{ref}`
- `build/daily-bible search <query>` — same as `GET /api/v1/search?q=...`
- `build/daily-bible random` — same as `GET /api/v1/random`
- `build/daily-bible <date>` — same as `GET /api/v1/date/{date}`
- Uses same `DB_PATH` env var as the server
- Outputs JSON to stdout

## API Layer

- All routes wrapped in `corsMiddleware` then `loggingMiddleware`
- **All DB queries use `QueryContext`/`QueryRowContext`** with `r.Context()` for cancellation
- `GET /api/v1/gospel/{ref}` — supports dot-separated non-contiguous segments (e.g. `Mt 5,20-22a.27-28`)
- `GET /api/v1/today` — queries `lectionary` table for today's reading (returns 404 if empty)
- `GET /api/v1/search?q=...` — phrase-only FTS search (query is wrapped in double quotes)
- `GET /api/v1/random` — retries up to 10 times on rowid gaps

## Package Structure

```
cmd/server/main.go          — HTTP server entrypoint
cmd/cli/main.go             — CLI entrypoint (`today`, `gospel`, `search`, `random`, date)
internal/api/               — handlers, router, middleware, ref parsing + query
internal/db/                — DB open + schema init
internal/parser/            — HTML parsing (extracted from tools/crawler)
internal/model/             — Gospel struct
internal/constants/         — DB path, server addr, crawler config
tools/crawler/              — Vatican News sitemap crawler
tools/tsv/                  — gospels.txt → TSV converter
data/                       — schema.sql, fts.sql, lectionary.sql, triggers.sql
test-data/                  — HTML fixtures for parser/crawler tests
```

## Testing

- Tests use `:memory:` SQLite databases with schema loaded from `data/schema.sql`
- Parser/crawler tests depend on HTML fixtures in `test-data/`
- `TestProcessedAndMissingWriters` uses `t.TempDir()` — do not revert to using real `build/` paths
- Crawler tests pass `sleepDuration=0` to `worker()` for instant execution

## CI (`.github/workflows/ci.yml`)

Order: `staticcheck` → `test-with-race-detector` → `make build`
