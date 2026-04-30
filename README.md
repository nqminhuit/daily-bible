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
curl 'http://localhost:8080/liveness'
curl 'http://localhost:8080/readiness'
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
.mode column
.headers on

SELECT book, chapter, verse, text FROM verses LIMIT 10;
SELECT text FROM verses_fts WHERE verses_fts MATCH 'Giêsu' LIMIT 10;
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
