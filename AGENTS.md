# AI Agent Instructions

This repository implements a small API for daily Gospel readings.

Before modifying code, read these files:

- architecture.md
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

## Unit tests

- Tests must be small and focus, avoid lengthy tests
- Use a real SQLite database for most tests
- Do NOT mock SQLite in most cases
- use memory mode for testing: `sql.Open("sqlite3", ":memory:")`
- Each test gets a new DB: `file:test.db?mode=memory&cache=shared`
- Use Test Fixtures, create helper (if not exist)
  ```
  testdb/
    schema.sql
    seed.sql
  ```
  then load:
  ```go
  func loadSchema(db *sql.DB) {
    schema, _ := os.ReadFile("testdata/schema.sql")
    db.Exec(string(schema))
  }
  ```
- Transaction Rollback Pattern if needed
- Create a test DB helper package (if not exist): `internal/testdb/testdb.go`
  ```go
  func New(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }

    loadSchema(db)

    return db
  }
  ```

---

## Database

- use SQLite
- use prepared statements
- keep schema in schema.sql

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
