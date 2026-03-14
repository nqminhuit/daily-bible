# Implementation Tasks

## Phase 1 — Project Setup

- [ ] initialize Go module
- [ ] create repository structure
- [ ] create schema.sql
- [ ] create DB initialization script
- [ ] unit tests for go module if any

---

## Phase 2 — Database Layer

- [ ] implement internal/db/db.go
- [ ] open SQLite connection
- [ ] load schema.sql
- [ ] create indexes
- [ ] unit tests for go module if any

---

## Phase 3 — Data Import

- generate SQL insert file from gospel.json (https://github.com/nqminhuit/liturgical-calendar/blob/master/resources/gospel.json)
- load SQL file into SQLite

Tasks:

- [ ] read gospel.json
- [ ] parse reading reference
- [ ] extract:
  - book
  - chapter
  - verse_start
  - verse_end
- [ ] add insert statement into `data/import_gospels.sql`

---

## Phase 4 — Full Text Search

- [ ] create gospels_fts virtual table
- [ ] add triggers
- [ ] implement search query:
    ```sql
    select reference
    from gospels_fts
    where gospels_fts match ?
    limit 20
    ```

---

## Phase 5 — API Server

Location:
```
cmd/server/
```

Tasks:

- [ ] initialize HTTP server
- [ ] create router
- [ ] load SQL `data/import_gospels.sql` into SQLite
- [ ] unit tests for go module if any

---

## Phase 6 — API Endpoints

### Get gospel by reference `GET /api/v1/gospel/{reference}`

- [ ] implement query gospels table by `reference`
- [ ] unit tests for go module if any

---

### Search `GET /api/v1/search?q={search_text}`

- [ ] implement FTS query
- [ ] unit tests for go module if any

---

### Random reading `GET /api/v1/random`

- [ ] implement random query
- [ ] unit tests for go module if any

---

## Phase 7 — Testing

- [ ] test importer
- [ ] test endpoints
- [ ] test FTS search

---

## Phase 8 — Deployment

- [ ] build binary
- [ ] test curl access
- [ ] verify API responses
