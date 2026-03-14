# AI Agent Instructions

This repository implements a small API for daily Gospel readings.

Before modifying code, read these files:

- architecture.md
- tasks.md
- coding-rules.md
- DECISIONS.md

Architectural constraints are defined in DECISIONS.md and must not be violated.
Implementation details must follow coding-rules.md.

---

## Development Workflow

1. Follow the architecture described in architecture.md.
2. Implement tasks sequentially from tasks.md.
3. Follow all rules in coding-rules.md.

---

## Implementation Strategy

When adding code:

- keep functions small
- prefer Go standard library
- avoid unnecessary dependencies
- code must be testable and have coverage

---

## Database

- use SQLite
- use prepared statements
- keep schema in schema.sql

---

## API

Endpoints:

```
GET /api/v1/gospel/{reference}
GET /api/v1/search?q=...
GET /api/v1/random
```

---

## Output Requirements

Code must:

- compile
- unit tests pass
- follow repository structure
- follow Go conventions
- return valid JSON

---

## External Knowledge

If implementation details are unclear or outdated:

1. Perform **web search**.
2. Verify the current documentation.
3. Use the most stable and widely used solution.

When to use **web search**:
- Searching the web for current information
- Finding recent documentation or updates
- Researching topics beyond your knowledge cutoff
- User requests information about recent events or current data
