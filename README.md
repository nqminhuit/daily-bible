# daily-bible

A Vietnamese daily Gospel reader with full-text search, powered by SQLite FTS5.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Data Pipeline                            │
│                                                                 │
│  Vatican News ──► tools/crawler ──► build/gospels.txt           │
│       sitemap              │                                    │
│                            ▼                                    │
│                      tools/tsv ──► build/gospels.tsv            │
│                                        │                        │
│                                        ▼                        │
│                                  make import-db                 │
│                                        │                        │
│                                        ▼                        │
│                                  resources/bible.db             │
│                                   (verses + FTS)                │
│                                                                 │
│  internal/lectionary ──► lectionarycrawler ──► daily_readings   │
│  (Easter algorithm,        (crawls Vatican      (date →         │
│   seasons, cycles)          for gospel refs)      gospel ref)   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      Application Layer                          │
│                                                                 │
│  ┌──────────────┐    ┌──────────────────────────────────────┐   │
│  │  cmd/server  │    │              internal/               │   │
│  │  (HTTP API)  │    │                                      │   │
│  │  :8090       │───►│  api/   router + handlers + ref      │   │
│  └──────────────┘    │  db/    Open + pragmas + migrations  │   │
│                      │  query/ FTS phrase query builder     │   │
│  ┌──────────────┐    │  model/ Gospel struct                │   │
│  │  cmd/cli     │    │  parser/ HTML parsing                │   │
│  │  (CLI tool)  │───►│  constants/ runtime defaults         │   │
│  └──────────────┘    │  lectionary/ liturgical calendar     │   │
│                      └──────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Package Structure

| Package | Purpose |
|---------|---------|
| `cmd/server` | HTTP server entrypoint, wires DB + router |
| `cmd/cli` | CLI entrypoint (`today`, `gospel`, `search`, `random`, date) |
| `internal/api` | HTTP handlers, router, middleware, reference compatibility wrappers |
| `internal/scripture` | Bounded reference parsing, normalization, verse queries and completeness |
| `internal/reading` | Scoped calendar imports, reviewed entry/date caches, observations and offline audits |
| `internal/db` | SQLite connection (WAL, single-connection pool), schema init |
| `internal/query` | FTS5 phrase query builder (wraps terms in double quotes) |
| `internal/model` | Gospel struct |
| `internal/parser` | HTML parsing for Vatican News pages |
| `internal/constants` | Runtime defaults: DB path, server address, timezone |
| `internal/crawlerconfig` | Crawler settings, Vatican URLs, pipeline paths shared with the TSV converter |
| `internal/lectionary` | Liturgical calendar (Easter via Meeus/Jones/Butcher, seasons, cycles) |
| `tools/crawler` | Vatican News sitemap crawler → `build/gospels.txt` |
| `tools/lectionarycrawler` | Cache-first resolution, bounded observations/reviews and offline coverage audits |
| `tools/calendar` | Pinned calendar-provider export/tests; maintenance-only Node dependency |
| `tools/tsv` | Converts `gospels.txt` → `build/gospels.tsv` |

## Data Pipeline Flow

```
Step 1: Crawl
  make crawler          # fetch 3 latest readings
  make crawler-all-urls # fetch all sitemap URLs

  Vatican News sitemap
       │
       ▼
  tools/crawler
       │  - fetches HTML pages
       │  - extracts Gospel text + verse numbers
       │  - tracks processed/failed URLs in build/
       ▼
  build/gospels.txt

Step 2: Transform
  make tsv

  build/gospels.txt
       │
       ▼
  tools/tsv
       │  - parses verse markers {{N}}
       │  - canonicalizes book codes (Mt, Mc, Lc, Ga)
       │  - deduplicates by book+chapter+verse
       ▼
  build/gospels.tsv

Step 3: Import
  make import-db

  build/gospels.tsv
       │
       ▼
  sqlite3 resources/bible.db
       │  - creates verses table (book, chapter, verse, verse_suffix, text)
       │  - creates verses_fts virtual table (unicode61, prefix indexing)
       │  - syncs FTS via triggers
       │  - creates daily_readings table
       ▼
  resources/bible.db  (ready to serve)

Step 4: Calendar-aware reading cache
  The annual job extends definitions and writes an OFFLINE coverage audit;
  it does not recrawl every generated date or approve predicted readings.
  See data/calendar-policy.md for the safe, temporary-copy operator workflow.

  tools/calendar (pinned provider; maintenance-only Node dependency)
       │  - exports selected observances, precedence, transfers, cycles
       │  - explicit calendar profile and version; no reading approvals
       ▼
  tools/lectionarycrawler
       │  - imports versioned calendar days; retains legacy estimates for audit
       │  - reuses reviewed date decisions / actual lectionary entries first
       │  - checks shared verse completeness before declaring availability
       │  - crawls gaps, explicit refreshes, and separate audit samples only
       ▼
  reading_calendar_days + reading_entries + retained reviews/observations
       │  - evidence scope and real source dates are preserved
       │  - warm supported windows, including future years: zero Gospel HTTP
       ▼
  API / CLI / gospel-site (same evidence contract; explicit calendar selection)
```

### Mass-reading correctness

Algorithmic `lectionary_key` values are **estimates**, not evidence of the Mass
reading or celebration name. Feasts, transferred solemnities, and local calendars
can override a normal weekday or Sunday. Vatican News's **Vietnamese dated page**
is the default reading source, but its language is **not proof that it follows a
Vietnamese diocesan or national calendar**. Parish-specific Masses, local calendar
rules, and permitted alternatives can legitimately differ. Confirm the intended
calendar before applying an archive-wide repair; unresolved source/calendar
conflicts must stay unavailable rather than silently replacing a local reading.

The default `READING_CALENDAR=vatican-vi` preserves the source-only contract:
exact-date evidence must match the current reference and actual dated page.
`roman-general` is an **opt-in evaluation profile**, not production approval.
The `vietnam` namespace requires reviewed custom national definitions:
automatic Romcal export/import and derived caches are blocked because the pinned
plugin misplaces the Vietnamese Martyrs. See [the investigation](data/calendar-research.md).
Supported calendars use versioned definitions and **explicitly reviewed reusable
entry mappings**, with date-specific reviewed choices taking precedence. Only the
`day` Mass option is currently supported; unspecified variants are not guessed.

Sharing verse text and supported lectionary mappings is intentional. Weekday
Gospels repeat annually across I/II first-reading cycles; Sunday Gospels retain
A/B/C. Version-2 exports declare `gospel_entry_schema: 2`; regenerate old manifests
and review changed dependencies rather than silently relabeling approvals.
Offline audits include a grouped, read-only `entries` review queue with source
observations and structural text diagnostics, not automatic candidate promotion.
What is forbidden is treating an estimated seasonal key, legacy majority, or arbitrary
source observation as proof of equivalent celebrations. Unreviewed references and
all observed disagreements remain available for audit. Missing/partial verse text
is reported separately rather than rendered as a complete Gospel.

Date/today responses identify `resolution_method` (`source_checked` or
`calendar_resolved`), `calendar_id`, `mass_option`, `entry_key`, `rules_version`,
`source_url`, `evidence_date`, `verified_at`, and the resolved/reviewed celebration.
A reused entry points to its **actual evidence**, not a fabricated page for the
requested future date. Compatibility `lectionary_key` values remain estimates.

Read [the policy and operator contract](data/calendar-policy.md) before importing
calendars, reviewing mappings, or changing a shipped snapshot. It covers temporary
copies, offline archive audits, `-review-entry`, `-accept-date`, scoped replacement,
and explicit refreshes. `DB_PATH` selects the audit copy. A normal warm-cache run
needs **zero Gospel HTTP requests**; unresolved entries fetch at most one reusable
observation candidate and still require a review, not automatic promotion.

Availability probes accept both supported evidence methods. Healing is bounded;
annual extension is offline; a separate source-audit workflow checks samples on a
disposable database. Missing-text repairs still use the verse import pipeline.
The separate `gospel-site` frontend reads its own SQLite snapshot directly, so
calendar settings, schema compatibility, coverage review, and release/rollback
snapshots must be coordinated across both repositories. **Nothing here authorizes
an archive-wide data promotion or deployment before those gates are satisfied.**

## Request Flow

```
                        ┌─────────┐
                        │ Client  │
                        └────┬────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
       ┌─────────────┐             ┌─────────────┐
       │  HTTP API   │             │  CLI tool   │
       │  cmd/server │             │  cmd/cli    │
       └──────┬──────┘             └──────┬──────┘
              │                           │
              └──────────┬────────────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │    internal/api     │
              │  router + handlers  │
              │  reading resolution │
              └──────────┬──────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │    internal/db      │
              │  SQLite (WAL mode)  │
              │  FTS5 search        │
              │  daily_readings     │
              │  verses             │
              └──────────┬──────────┘
                         │
                         ▼
                   JSON response
```

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/gospel/{ref}` | Get verses by reference (e.g. `Ga 9,1-41`, `Mt 5,20-22a.27-28`) |
| `GET /api/v1/search?q=...` | Full-text phrase search |
| `GET /api/v1/random` | Random verse |
| `GET /api/v1/today` | Today's Gospel reading (requires lectionary) |
| `GET /api/v1/date/{date}` | Gospel reading for a specific date (requires lectionary) |
| `GET /liveness` | Health check |
| `GET /readiness` | Readiness check |

Gospel references support suffixes (`Ga 1,1a-3b`), dot-separated ranges
(`Mt 5,20-22a.27-28`), and forward cross-chapter ranges (`Mt 18,21-19,1`).
Cross-chapter ranges include intervening chapters. Chapter and verse numbers
must be positive integers; malformed or reversed ranges are rejected. References
are limited to 1,024 decoded bytes (including whitespace) and 32 ranges after
cross-chapter expansion. A cross-chapter range counts once per included chapter.
Invalid or over-limit references return HTTP 400; valid references with no
matching verses return 404, and database failures return 500. The CLI uses the
same reference validation.

## How to use

### Prerequisites

- Go 1.25+
- `sqlite3` CLI

Install sqlite3 (Ubuntu/Debian):

```shell
sudo apt-get install sqlite3
```

### 1) Build the local Bible database

Run the full ingestion pipeline:

```shell
make crawler        # crawl latest readings (use make crawler-all-urls for full crawl)
make tsv            # convert build/gospels.txt -> build/gospels.tsv
make import-db      # create build/bible.db and load schema + FTS
```

By default the database is created at `build/bible.db`. Override with `DB_PATH`:

```shell
DB_PATH=/path/to/bible.db make import-db
```

Crawler state files are kept under `build/`:
- `processed.txt`: URLs successfully parsed
- `failed.txt`: URLs that failed parsing and are skipped on next runs
- `missing_verse_number.txt`: extracted pages without verse `<sup>` markers

If you want to retry previously failed URLs after parser updates, delete `build/failed.txt` before running crawler again.

### 1b) Lectionary

The annual GitHub job extends calendar definitions and records an **offline** audit.
It does not approve predicted references or crawl every date. Calendar imports and
reviews are separate operations; follow [the operator workflow](data/calendar-policy.md)
on a temporary database before proposing a snapshot change.

The low-level command below generates estimates and fills gaps under the selected
`READING_CALENDAR` (source-only unless explicitly configured). Prefer bounded
`make verify-readings FROM=... TO=...` for a targeted investigation:

```shell
# Single year:
make crawl-lectionary YEARS="2026"

# Multiple years:
make crawl-lectionary YEARS="2025,2026,2027"
```

Imports and identical reviews are idempotent. Existing Gospel references are
retained; matching old keys never authorize copying a reference.

### 2) Start the API server

```shell
make dev
# or with a custom port:
make dev PORT=:8080
# or with a custom database:
DB_PATH=/path/to/bible.db make dev
```

Server runs on `http://localhost:8090` by default.

### 3) Use the CLI tool

Build the CLI:

```shell
make build-cli
# or build manually:
go build -tags "fts5" -o build/daily-bible ./cmd/cli
```

The CLI outputs JSON to stdout:

```shell
./build/daily-bible today
./build/daily-bible 2026-03-22
./build/daily-bible gospel "Ga 9,1-41"
./build/daily-bible search "Chúa Giê-su"
./build/daily-bible random
./build/daily-bible help
```

Override the database path:

```shell
DB_PATH=/path/to/bible.db ./build/daily-bible today
```

### 4) Call API endpoints

```shell
curl 'http://localhost:8090/api/v1/gospel/Ga%209,1-41'
curl 'http://localhost:8090/api/v1/search?q=Ch%C3%BAa+Gi%C3%AA-su'
curl 'http://localhost:8090/api/v1/random'
curl 'http://localhost:8090/api/v1/today'
curl 'http://localhost:8090/api/v1/date/2026-03-22'
curl 'http://localhost:8090/liveness'
curl 'http://localhost:8090/readiness'
```

#### Today/Date Response Format

```json
{
  "date": "2026-03-22",
  "lectionary_key": "lent_5_sun_A",
  "ref": "Ga 11,1-45",
  "verses": "verse one text verse two text ..."
}
```

## Query the SQLite database

Open DB:

```shell
sqlite3 build/bible.db
```

Useful sqlite commands:

```sql
.tables
.schema verses
.schema verses_fts
.schema daily_readings
.mode column
.headers on

SELECT book, chapter, verse, text FROM verses LIMIT 10;
SELECT text FROM verses_fts WHERE verses_fts MATCH 'Giêsu' LIMIT 10;
SELECT * FROM daily_readings LIMIT 5;
SELECT * FROM daily_readings WHERE date = '2026-03-22';
```

Run query directly from shell:

```shell
sqlite3 build/bible.db "SELECT book, chapter, verse FROM verses LIMIT 5;"
```

## Development commands

```shell
make help
make fmt
make test
make test-with-race-detector
make compile
make build
```

## Makefile targets

| Target | Description |
|--------|-------------|
| `make crawler` | Crawl 3 latest Bible readings |
| `make crawler-all-urls` | Crawl all sitemap URLs |
| `make tsv` | Convert gospels.txt to TSV format |
| `make import-db` | Import data into SQLite database |
| `make crawl-lectionary YEARS=2026,2027` | Generate estimates and resolve/fill gaps under the explicitly selected reading policy |
| `make verify-readings FROM=2026-11-09 TO=2026-11-11` | Verify a bounded date window against dated Vatican News pages |
| `make setup-lectionary YEARS=2026,2027` | Full lectionary setup (alias for crawl-lectionary) |
| `make build` | Build server binary (`build/daily-bible-server`) |
| `make build-cli` | Build CLI binary (`build/daily-bible`) |
| `make dev` | Run server in development mode (default `:8090`) |
| `make dev PORT=:8080` | Run server on custom port |
| `make clean` | Clean build artifacts |
