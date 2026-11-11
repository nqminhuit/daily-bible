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
| Dev server | `make dev` (listens on `:8090`, override with `make dev PORT=:8080`) |
| Populate daily readings | `make crawl-lectionary YEARS="2026,2027"` |
| CI lint | `staticcheck ./...` |

## Data Pipeline

```
Vatican News sitemap → tools/crawler → build/gospels.txt → tools/tsv → build/gospels.tsv → make import-db → resources/bible.db
Legacy estimates (internal/lectionary) → daily_readings (audit/compatibility only)
Pinned tools/calendar export → versioned calendar days + explicitly reviewed entries → cache-first lookups
```

- Crawler state lives in `build/` (`processed.txt`, `failed.txt`, `missing_verse_number.txt`)
- To retry failed URLs after parser changes: delete `build/failed.txt` before re-running crawler
- `build/` is gitignored — never commit generated artifacts
- `tools/crawler` and `tools/tsv` do **not** need `-tags "fts5"` (no SQLite dependency); only `tools/lectionarycrawler` does

## Database

- **Runtime path**: `resources/bible.db`, overridable via `DB_PATH` env var
- **Schema**: `verses(book, chapter, verse, verse_suffix, text)` — composite PK includes `verse_suffix`
- **Daily readings**: `daily_readings(date TEXT PRIMARY KEY, lectionary_key TEXT, gospel_ref TEXT)` — calendar keys are estimates; legacy refs may be unverified
- **Policy**: read `data/calendar-policy.md` and `data/calendar-research.md`. The owner approved Vietnam national/daytime/no assumed local overlay as the target on 2026-11-10: follow an authoritative Gospel assignment, otherwise the approved lectionary's first-listed full authorized option. This is not provider/data/deployment approval. `READING_CALENDAR` still defaults to source-only `vatican-vi`; `roman-general` is evaluation-only. Automatic `vietnam` Romcal exports/imports/reusable reviews and derived lookups are blocked (Vietnamese Martyrs date/rank discrepancy); reviewed national definitions need their own rules version; diocesan/parish profiles need a distinct implemented namespace. Only `READING_MASS_OPTION=day` is supported.
- `data/vietnam-national-sources.json` is a partial source dossier, not an executable calendar or Gospel approval. The explicit `node tools/calendar/vietnam-pilot.cjs 2026` pilot supports nine sourced definitions and marks 356 dates unsupported; all other years are rejected. Automatic Vietnam export stays blocked. Read `data/vietnam-pilot-audit.md`; never activate this incomplete pilot in production. Its Ash Mass transfer does not transfer Lent or the Hours. The national proclamation lists both Ascension schedules; do not infer their territorial/default applicability.
- `data/vietnam-content-review.json` / `data/vietnam-content-audit.md` record the nine-reading temporary-copy continuation, the first-listed Mồng Ba Tết review (`Mt 25,14-30`, lectionary pages not inspected), diocesan rule findings that are evidence only, and the reviewed missing-only five-verse repair. November 15 (optional Martyrs celebration) and Ascension (Thursday or Sunday by diocese) need an owner default, not more searching. The found Vĩnh Long Ordo is diocesan and the 2017 lectionary catalogue is not approval/frontmatter; neither establishes the full national source contract. Evidence inventories never auto-approve/import data.
- **Provenance**: `daily_reading_sources` retains exact-date source-only evidence. Migration 004 adds calendar definitions, reusable entries, observations, reviews and date decisions; migration 005 retains exact per-day rule evidence, underlying observance/seasons, explicit Mass assignments and unsupported reasons. Pilot warm caches/reviews require matching definition evidence. Never silently promote legacy refs or observations.
- `tools/lectionarycrawler` applies migrations 003/004/005 idempotently. Use a consistent temporary copy and `-offline -audit-report=...` for archive audits. The grouped `entries` queue contains candidates, not approvals. Warm supported windows must use zero Gospel HTTP requests.
- Weekday Gospel-only entries repeat annually across I/II; Sunday entries retain A/B/C and proper/memorial choices stay separate. Manifests require `gospel_entry_schema: 3`; regenerate old definitions and explicitly re-review dependencies, never relabel old approvals.
- `-review-entry` explicitly approves a reusable mapping; `-accept-date` approves only that date/calendar/option. Replacements require `-replace-entry`/`-replace-date` and retain history. Legacy source-only corrections require `-accept-corrections`.
- Unreviewed assigned national Masses need formulary evidence, not an automatic wrong-scope Vatican crawl. Explicit refresh of a supported entry remains a separate source audit.
- Never reuse by the old `lectionary_key`, invent target-date evidence, or invalidate approved local selections because a different-scope source disagrees. A stale date-specific decision blocks ordinary-entry fallback. Missing verse text needs a verse-store repair, not repeated reference crawling.
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

- All routes wrapped in `loggingMiddleware(corsMiddleware(mux))` — logging is outermost
- **All DB queries use `QueryContext`/`QueryRowContext`** with `r.Context()` for cancellation
- `GET /api/v1/gospel/{ref}` — supports dot-separated non-contiguous segments (e.g. `Mt 5,20-22a.27-28`)
- `GET /api/v1/today` — requires supported scoped evidence (`source_checked` or `calendar_resolved`) and complete verse text; date endpoints/CLI use the same contract. Returns 404 for unresolved, stale, conflicting, or incomplete readings; labels come from reviewed dates/resolved observances, never seasonal estimates.
- `GET /api/v1/search?q=...` — phrase-only FTS search (query is wrapped in double quotes)
- `GET /api/v1/random` — retries up to 10 times on rowid gaps

## Package Structure

```
cmd/server/main.go          — HTTP server entrypoint
cmd/cli/main.go             — CLI entrypoint (`today`, `gospel`, `search`, `random`, date)
internal/api/               — handlers, router, middleware, compatibility reference wrappers
internal/scripture/         — bounded shared reference parsing, normalization, querying, text completeness
internal/db/                — DB open + schema init (ApplyMigrations used in tests only; server assumes DB exists)
internal/query/             — FtsPhraseQuery: wraps FTS search terms in double quotes
internal/parser/            — HTML parsing (extracted from tools/crawler)
internal/model/             — Gospel struct
internal/reading/           — scoped calendar imports, reviewed caches, date decisions, observations, audits
internal/constants/         — runtime defaults: DB path, server addr, timezone
internal/crawlerconfig/     — crawler settings, Vatican URLs, crawler/TSV artifact paths
internal/lectionary/        — Liturgical calendar generation (Easter, seasons, lectionary keys)
tools/crawler/              — Vatican News sitemap crawler
tools/lectionarycrawler/    — cache-first maintenance, offline audits, bounded source observations/reviews
tools/calendar/             — pinned Node calendar provider; export/test only, never a Go runtime dependency
tools/tsv/                  — gospels.txt → TSV converter
data/                       — migration SQL files
test-data/                  — HTML fixtures for parser/crawler tests
```

## Testing

- Tests use `:memory:` SQLite databases with schema loaded from `data/migrations/`
- Parser/crawler tests depend on HTML fixtures in `test-data/`
- `TestProcessedAndMissingWriters` uses `t.TempDir()` — do not revert to using real `build/` paths
- Crawler tests pass `sleepDuration=0` to `worker()` for instant execution

## CI (`.github/workflows/ci.yml`)

Order: `staticcheck` → `test-with-race-detector` → `make build`
