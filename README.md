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

Crawler state files are kept under `build/`:
- `processed.txt`: URLs successfully parsed
- `failed.txt`: URLs that failed parsing and are skipped on next runs
- `missing_verse_number.txt`: extracted pages without verse `<sup>` markers

If you want to retry previously failed URLs after parser updates, delete `build/failed.txt` before running crawler again.

### 1b) Set up the lectionary (optional)

The lectionary system maps liturgical dates to Gospel readings. It requires:
1. A liturgical calendar JSON file (one per year)
2. A crawl of Vatican News to resolve lectionary keys to Gospel references

```shell
# 1) Download calendar JSON (e.g. for 2026):
make import-calendar FILE=resources/liturgical-calendar-2026.json
# or fetch directly from GitHub:
make import-calendar URL=https://raw.githubusercontent.com/nqminhuit/liturgical-calendar/refs/heads/master/resources/liturgical-calendar-2026.json

# 2) Crawl lectionary keys:
make crawl-lectionary

# or run both at once:
make setup-lectionary URL=https://raw.githubusercontent.com/nqminhuit/liturgical-calendar/refs/heads/master/resources/liturgical-calendar-2026.json
```

Calendar JSON files should be placed under `resources/`:
```
resources/
  liturgical-calendar-2025.json
  liturgical-calendar-2026.json
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
```

Server runs on `http://localhost:8080`.

### 3) Call API endpoints

```shell
curl 'http://localhost:8080/api/v1/gospel/Ga%209,1-41'
curl 'http://localhost:8080/api/v1/search?q=Ch%C3%BAa+Gi%C3%AA-su'
curl 'http://localhost:8080/api/v1/random'
curl 'http://localhost:8080/api/v1/today'
curl 'http://localhost:8080/api/v1/date/2026-03-22'
curl 'http://localhost:8080/liveness'
curl 'http://localhost:8080/readiness'
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
  "verses": [
    {"book": "Ga", "chapter": 11, "verse": 1, "text": "..."},
    {"book": "Ga", "chapter": 11, "verse": 2, "text": "..."}
  ]
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
.schema lectionary
.schema calendar
.mode column
.headers on

SELECT book, chapter, verse, text FROM verses LIMIT 10;
SELECT text FROM verses_fts WHERE verses_fts MATCH 'Giêsu' LIMIT 10;
SELECT * FROM lectionary LIMIT 5;
SELECT * FROM calendar WHERE date = '2026-03-22';
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
| `make import-calendar FILE=...` | Import liturgical calendar JSON |
| `make crawl-lectionary` | Populate lectionary table from Vatican News |
| `make setup-lectionary FILE=...` | Full lectionary setup (calendar + crawl) |
| `make build` | Build server binary |
| `make dev` | Run server in development mode |
| `make clean` | Clean build artifacts |
