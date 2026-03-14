# Architecture: Liturgical Daily Bible API

## Overview

This project provides a lightweight API that serves daily Gospel readings.

Design goals:

- lightweight
- self-hosted
- minimal dependencies
- Go standard library where possible

Clients may include:

- browser
- bash / curl / postman

---

## 1. System Architecture

```
SQLite database
   ↓
Go HTTP API server
   ↓
Clients
```

---

## 2. Technology Stack

Language:
- Go

Database:
- SQLite

Search:
- SQLite FTS5

Web server:
- Go net/http

---

## 3. Repository Structure

```
daily-bible-api/
    Makefile
    cmd/
        server/
            main.go
    internal/
        db/
            db.go
        model/
            gospel.go
            calendar.go
        api/
            router.go
            handlers.go
        search/
            fts.go
    data/
        import_gospels.sql
        readings.db
        schema.sql
```

## 4. SQLite Schema

Create schema file:

```
data/schema.sql
```

### Main table

```sql
CREATE TABLE gospels (
    id INTEGER PRIMARY KEY,
    reference TEXT UNIQUE NOT NULL,
    book TEXT,
    chapter_start INTEGER,
    verse_start INTEGER,
    chapter_end INTEGER,
    verse_end INTEGER,
    text TEXT NOT NULL
);
```

Example row:

```
reference: "Mt 26,14-27,66"
book: Mt
chapter_start: 26
verse_start: 14
chapter_end: 27
verse_end: 66
text: "..."
```

### Indexes

```sql
CREATE INDEX idx_gospel_book
ON gospels(book);
```

## 5. Full Text Search

Use **SQLite FTS5**.

```sql
CREATE VIRTUAL TABLE gospels_fts
USING fts5(
  reference,
  text,
  content='gospels',
  content_rowid='id'
);
```

Triggers:

```sql
CREATE TRIGGER gospels_ai AFTER INSERT ON gospels BEGIN
  INSERT INTO gospels_fts(rowid, reference, text)
  VALUES (new.id, new.reference, new.text);
END;

CREATE TRIGGER gospels_ad AFTER DELETE ON gospels BEGIN
  INSERT INTO gospels_fts(gospels_fts, rowid, reference, text)
  VALUES('delete', old.id, old.reference, old.text);
END;

CREATE TRIGGER gospels_au AFTER UPDATE ON gospels BEGIN
  INSERT INTO gospels_fts(gospels_fts, rowid, reference, text)
  VALUES('delete', old.id, old.reference, old.text);
  INSERT INTO gospels_fts(rowid, reference, text)
  VALUES(new.id, new.reference, new.text);
END;
```

## 6. API Server

Location:

```
cmd/server/main.go
```

Use **Go standard library**.

Router:

```
net/http
```

## 7. API Endpoints

### 1. ### Get gospel by book, chapter and verse start, end 

```
GET /api/v1/gospel/{reference}
```

Example:

```
/api/v1/gospel/Mt%2026,14-27,66
```

### 2. Search text

```
GET /api/v1/search?q=Giêsu
```

Query:

```
SELECT reference
FROM gospels_fts
WHERE gospels_fts MATCH ?
LIMIT 20
```

### 3. Random reading 

```
GET /api/v1/random
```

SQL:

```
SELECT reference, text
FROM gospels
ORDER BY RANDOM()
LIMIT 1;
```

## 8. Example Handler

```go
func GetGospelByReference(w http.ResponseWriter, r *http.Request) {
    ref := r.PathValue("ref")

    row := db.QueryRow(`
        SELECT reference, text
        FROM gospels
        WHERE reference = ?
    `, ref)

    var g Gospel
    err := row.Scan(&g.Reference, &g.Text)

    if err != nil {
        http.Error(w, "not found", 404)
        return
    }

    json.NewEncoder(w).Encode(g)
}
```

## 9. CLI Usage

### Bash

```
curl http://localhost:8080/api/v1/search?q=Giêsu
curl http://localhost:8080/api/v1/gospel/Mt%2026,14-27,66
curl http://localhost:8080/api/v1/random
```
