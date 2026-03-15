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

## System Architecture

```
SQLite database
   ↓
Go HTTP API server
   ↓
Clients
```

---

## Technology Stack

Language:
- Go

Database:
- SQLite

Search:
- SQLite FTS5

Web server:
- Go net/http

---

## SQLite Schema

Schema:

```
data/schema.sql
```

### Main table

```sql
-- Each row = one verse
CREATE TABLE IF NOT EXISTS verses (
    book TEXT NOT NULL,
    chapter INTEGER NOT NULL,
    verse INTEGER NOT NULL,
    text TEXT NOT NULL,
    PRIMARY KEY(book, chapter, verse)
);
```

Example row:

```
Ga|9|41|Đức Giê-su bảo họ: “Nếu các ông đui mù, thì các ông đã chẳng có tội. Nhưng giờ đây các ông nói rằng: ‘Chúng tôi thấy’, nên tội các ông vẫn còn!”
```

## Full Text Search

- Use **SQLite FTS5**.

- Use triggers on insert, update, delete to sync with the source table

## API Server

- Location:
  ```
  cmd/server/main.go
  ```

- Use **Go standard library**.

- Router:
  ```
  net/http
  ```

## API Endpoints

### Get gospel by reference

```
GET /api/v1/gospel/{reference}
```

Example:

```
/api/v1/gospel/Mt%2026,14-27,66
```

### Search text

```
GET /api/v1/search?q=Giesu
```

### Random verse

```
GET /api/v1/random
```

## CLI Usage

``` bash
curl http://localhost:8080/api/v1/search?q=Giêsu
curl http://localhost:8080/api/v1/gospel/Mt%2026,14-27,66
curl http://localhost:8080/api/v1/random
```
