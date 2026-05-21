# Copilot instructions for `daily-bible`

## Build, test, lint commands

- Show available tasks: `make help`
- Format: `make fmt`
- Full unit tests (with FTS5 tag): `make test`
- Race tests (CI-style): `make test-with-race-detector`
- Run a single test:
  - `go test -tags "fts5" ./internal/api -run TestSearchHandler_SuccessFTS`
  - `go test -tags "fts5" ./tools/crawler -run TestExtractGospelFromFixture`
- Compile all packages: `make compile`
- Build server binary: `make build`
- Lint (same tool as CI): `go install honnef.co/go/tools/cmd/staticcheck@latest && staticcheck ./...`
- Local API smoke checks (from README, after `make dev`):
  `curl 'http://localhost:8090/api/v1/gospel/Ga%209,1-41'`
  `curl 'http://localhost:8090/api/v1/search?q=Ch%C3%BAa+Gi%C3%AA-su'`
  `curl 'http://localhost:8090/api/v1/random'`
  `curl 'http://localhost:8090/liveness'`
  `curl 'http://localhost:8090/readiness'`

## High-level architecture

1. **Data ingestion pipeline** (`tools/crawler` -> `tools/tsv` -> SQLite import)
   - `tools/crawler` fetches Vatican News sitemap URLs, extracts Gospel content/reference from HTML, and writes `build/gospels.txt`.
   - `tools/tsv` parses `build/gospels.txt`, extracts `(book, chapter, verse, text)`, de-duplicates by `(book, chapter, verse)`, and writes `build/gospels.tsv`.
   - `make import-db` recreates `build/bible.db`, loads `data/schema.sql`, imports TSV into `verses`, creates FTS (`data/fts.sql`), rebuilds index, and installs triggers (`data/triggers.sql`).

2. **Database layer** (`internal/db`)
   - `Open()` applies SQLite PRAGMAs (`WAL`, `synchronous=NORMAL`, `foreign_keys=ON`) and constrains the pool to a single connection.
   - Runtime DB path is centralized in `internal/constants.DBPath` (`build/bible.db`).

3. **HTTP API layer** (`cmd/server` + `internal/api`)
   - `cmd/server/main.go` opens DB, builds router, and runs `net/http` server (default `:8090`, override with `-port`).
   - `NewRouter()` computes `MAX(rowid)` at startup and wires:
     - `GET /api/v1/gospel/{reference}`
     - `GET /api/v1/search?q=...`
     - `GET /api/v1/random`
     - `GET /liveness`
     - `GET /readiness`

4. **Schema and search model**
   - Primary table: `verses(book, chapter, verse, text)` with composite primary key.
   - Search uses `verses_fts` (FTS5, `unicode61 remove_diacritics 2`, prefix indexing) synced by triggers on insert/update/delete.

## Key conventions in this repository

- **Always use the FTS5 build tag** when testing/building Go code. The repo Makefile sets `GOFLAGS=-tags "fts5"` for this reason.
- **Reference format is strict** across API + tooling: short book code and chapter/range like `Ga 10,31-42`.
- **`build/` is the operational workspace** for generated artifacts (`gospels.txt`, `gospels.tsv`, `bible.db`, progress files). Avoid committing generated outputs.
- **Crawler/parser logic is Vatican-HTML specific** and keyed to Catholic liturgical markers (for example `Tin Mừng`, `Lời Chúa`, `✠`) and Vietnamese text/diacritics; preserve this behavior when changing parsing.
- **Random verse endpoint assumes immutable/dense rowids** (`SELECT ... WHERE rowid >= ? LIMIT 1`); changes to import strategy that break rowid density impact `/api/v1/random`.
- **Tests use real SQLite** (often `:memory:`) with schema loaded from `data/schema.sql`; parser/crawler tests rely on fixtures in `test-data/`.
