# Architectural Decisions

This document records important design decisions for this project.

These decisions should not be changed unless there is a strong technical reason.

---

# Use Go Standard Library HTTP Server

Decision: Use the Go `net/http` package instead of frameworks.

Reason:
- minimal dependencies
- small binary
- easier deployment
- sufficient for this API

Rejected alternatives:
- Gin
- Echo
- Fiber

---

# Use SQLite

Decision: Use SQLite as the primary database.

Reason:
- single-file database
- simple self-hosting
- low resource usage
- sufficient for read-heavy workloads

Rejected alternatives:
- PostgreSQL
- MongoDB

---

# Use SQLite FTS5 for Search

Decision: Use SQLite built-in FTS5 for text search.

Reason:
- avoids external search services
- integrated with database
- simple deployment

Rejected alternatives:
- Elasticsearch
- Meilisearch
- external search engines

---

# Use SQLite WAL Mode

Decision: Enable SQLite WAL (Write-Ahead Logging) mode.

Reason:
- improves concurrent reads
- prevents reader/writer blocking
- suitable for read-heavy APIs

Implementation: `PRAGMA journal_mode=WAL`

---

# SQLite Connection Pool Strategy

Decision: Use a single database connection.

Reason: SQLite is file-based and multiple connections can cause locking.

Implementation:
```go
db.SetMaxOpenConns(1)
db.SetMaxIdleConns(1)
```

---

# Keep API Simple

Decision: Expose a small REST API with a few endpoints.

Endpoints:
```
GET /api/v1/gospel/{reference}
GET /api/v1/search?q=...
GET /api/v1/random
```

Reason:
- easy to consume from CLI tools
- easy to integrate
- stable interface

---

# Keep Binary Small

Decision: Compile a single static Go binary.

Build command:
```
go build -ldflags="-s -w" ./cmd/server
```

Reason:
- easy deployment
- minimal runtime requirements
- suitable for small servers

---

# No Heavy Frameworks

Decision: Avoid heavy frameworks unless absolutely required.

Reason:
- keeps the project lightweight
- reduces dependency maintenance
- simplifies long-term support
