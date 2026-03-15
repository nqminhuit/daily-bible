# Start the project with agentic AI

Prompt:

```shell
Read AGENTS.md and all referenced documents and follow tasks.md and implement tasks sequentially.
```

# Query DB

Install sqlite if not:

``` shell
sudo apt-get install sqlite3
```

Start:

``` shell
sqlite3 build/bible.db
```

``` sql
.tables -- list tables
.schema gospels -- show schema
.databases -- show loaded DBs

-- Pretty Output in Terminal
.mode column
.headers on

SELECT * FROM gospels LIMIT 10;
SELECT reference, text FROM gospels WHERE reference = 'Ga 10,31-42';

-- Query Full-Text Search
SELECT reference FROM gospels_fts WHERE gospels_fts MATCH 'Giêsu';
```

query from bash directly:

``` shell
sqlite3 readings.db "SELECT reference FROM gospels LIMIT 5;"
```

# Development

## Start Server

``` shell
make import-sqlite
make dev
```

## E2E tests

``` shell
curl 'http://localhost:8080/api/v1/gospel/Ga%2010,31-42'
curl 'http://localhost:8080/api/v1/search?q=Ch%C3%BAa+Gi%C3%AA-su'
curl 'http://localhost:8080/api/v1/random'
```

# Crawler

``` shell
go run tools/biblelinks/main.go
go run tools/crawler/main.go
go run tools/tsv/main.go
```
