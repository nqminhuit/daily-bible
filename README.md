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
│  │  (CLI tool)  │───►│  constants/ paths, URLs, config      │   │
│  └──────────────┘    │  lectionary/ liturgical calendar     │   │
│                      └──────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Package Structure

| Package | Purpose |
|---------|---------|
| `cmd/server` | HTTP server entrypoint, wires DB + router |
| `cmd/cli` | CLI entrypoint (`today`, `gospel`, `search`, `random`, date) |
| `internal/api` | HTTP handlers, router, middleware, reference parsing + query |
| `internal/db` | SQLite connection (WAL, single-connection pool), schema init |
| `internal/query` | FTS5 phrase query builder (wraps terms in double quotes) |
| `internal/model` | Gospel struct |
| `internal/parser` | HTML parsing for Vatican News pages |
| `internal/constants` | DB path, server address, crawler config |
| `internal/lectionary` | Liturgical calendar (Easter via Meeus/Jones/Butcher, seasons, cycles) |
| `tools/crawler` | Vatican News sitemap crawler → `build/gospels.txt` |
| `tools/lectionarycrawler` | Generates `daily_readings` from algorithmic calendar + Vatican crawl |
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

Step 4: Lectionary (optional)
  make crawl-lectionary YEARS="2026,2027"

  internal/lectionary
       │  - computes Easter (Meeus/Jones/Butcher)
       │  - determines liturgical seasons + week numbers
       │  - generates lectionary keys (e.g. lent_5_sun_A)
       ▼
  tools/lectionarycrawler
       │  - maps lectionary keys → Vatican News URLs
       │  - crawls to extract gospel_ref (e.g. "Ga 11,1-45")
       ▼
  daily_readings table  (date → lectionary_key → gospel_ref)
```

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
              │  ref parsing        │
              └──────────┬──────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │    internal/db      │
              │  SQLite (WAL mode)  │
              │  FTS5 search        │
              │  daily_readings     │
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

### 1b) Set up the lectionary (optional)

The lectionary system maps liturgical dates to Gospel readings, then crawls Vatican News to resolve them to Gospel references (biblical book/chapter/verse).

Dates are generated algorithmically using the built-in `internal/lectionary` package (Easter computed via the Meeus/Jones/Butcher algorithm, seasons, cycles, and week numbers). No external calendar data is needed.

```shell
# Single year:
make crawl-lectionary YEARS="2026"

# Multiple years:
make crawl-lectionary YEARS="2025,2026,2027"

# Alias:
make setup-lectionary YEARS="2026,2027"
```

Re-running is idempotent (`INSERT OR IGNORE`).

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
| `make crawl-lectionary YEARS=2026,2027` | Populate daily_readings via algorithmic calendar generation |
| `make setup-lectionary YEARS=2026,2027` | Full lectionary setup (alias for crawl-lectionary) |
| `make build` | Build server binary (`build/daily-bible-server`) |
| `make build-cli` | Build CLI binary (`build/daily-bible`) |
| `make dev` | Run server in development mode (default `:8090`) |
| `make dev PORT=:8080` | Run server on custom port |
| `make clean` | Clean build artifacts |
