# daily-bible

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

The lectionary system maps liturgical dates to Gospel readings. It fetches a liturgical calendar JSON (one per year) and crawls Vatican News to resolve lectionary keys to Gospel references.

`CALENDAR_URLS` is a comma-separated list of URLs — one per year. Re-running is idempotent (`INSERT OR IGNORE`):

```shell
# Single year:
make crawl-lectionary CALENDAR_URLS="https://raw.githubusercontent.com/nqminhuit/liturgical-calendar/refs/heads/master/resources/liturgical-calendar-2026.json"

# Multiple years (comma-separated, no spaces):
make crawl-lectionary CALENDAR_URLS="https://raw.githubusercontent.com/nqminhuit/liturgical-calendar/refs/heads/master/resources/liturgical-calendar-2025.json,https://raw.githubusercontent.com/nqminhuit/liturgical-calendar/refs/heads/master/resources/liturgical-calendar-2026.json,https://raw.githubusercontent.com/nqminhuit/liturgical-calendar/refs/heads/master/resources/liturgical-calendar-2027.json"

# Alias:
make setup-lectionary CALENDAR_URLS="https://raw.githubusercontent.com/nqminhuit/liturgical-calendar/refs/heads/master/resources/liturgical-calendar-2026.json"
```

The calendar JSON format (from [nqminhuit/liturgical-calendar](https://github.com/nqminhuit/liturgical-calendar)):
```json
{
  "2026-03-22": {
    "lectionary_key": "lent_5_sun_A",
    "season": "lent",
    "sunday_cycle": "A",
    "week_of_season": 5,
    "weekday": "sun",
    "weekday_cycle": ""
  }
}
```

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

#### Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/v1/gospel/{ref}` | Get verses by reference (e.g. `Ga 9,1-41`, `Mt 5,20-22a.27-28`) |
| `GET /api/v1/search?q=...` | Full-text phrase search |
| `GET /api/v1/random` | Random verse |
| `GET /api/v1/today` | Today's Gospel reading (requires lectionary setup) |
| `GET /api/v1/date/{date}` | Gospel reading for a specific date (ISO 8601, requires lectionary setup) |
| `GET /liveness` | Health check |
| `GET /readiness` | Readiness check |

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
| `make crawl-lectionary CALENDAR_URLS=...` | Populate daily_readings (comma-separated URLs, one per year) |
| `make setup-lectionary CALENDAR_URLS=...` | Full lectionary setup (alias for crawl-lectionary) |
| `make build` | Build server binary (`build/daily-bible-server`) |
| `make build-cli` | Build CLI binary (`build/daily-bible`) |
| `make dev` | Run server in development mode (default `:8090`) |
| `make dev PORT=:8080` | Run server on custom port |
| `make clean` | Clean build artifacts |
