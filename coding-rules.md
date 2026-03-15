# Coding Rules

## Language

Use Go.

Prefer Go standard library whenever possible.

Avoid heavy frameworks.

---

## Dependencies

Allowed:

- SQLite driver
- standard library

Avoid unnecessary external dependencies.

---

## Project Structure

Follow the repository layout defined in architecture.md.

Responsibilities:

cmd/
- application entry points

internal/
- application logic

data/
- static resources like database file, sql files

tools/
- tools needed to prepare data for sqlite

Makefile: defined targets that help development easier
- unit tests (with coverage): `make test`
- compile go code: `make compile`
- import data to Sqlite (this should include init db if not exist): `make import-db`
- build the binary server: `make build` (should include compile and test)

---

## Database Access

- all database logic lives in internal/db
- SQL statements should be readable
- prefer prepared statements

---

## SQLite Configuration

The database must be configured at startup.

Required pragmas:

PRAGMA journal_mode=WAL
PRAGMA synchronous=NORMAL
PRAGMA foreign_keys=ON

---

## SQLite Connection Pool

SQLite must use a single connection.

Example configuration:

db.SetMaxOpenConns(1)
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)

---

## HTTP API

Use: net/http

Router should remain simple.

Do not introduce large web frameworks.

---

## Error Handling

- return clear HTTP errors
- log server errors
- do not panic in handlers

---

## JSON

Use: encoding/json

Responses must be valid JSON.

---

## Performance

- limit search results
- avoid loading unnecessary data
- reuse DB connections

---

## Security

- limit query length
- set HTTP timeouts
- validate inputs
