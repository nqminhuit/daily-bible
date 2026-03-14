package db

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"strings"
)

// Open opens a SQLite database at path and applies required PRAGMAs.
func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("empty database path")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Apply pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma failed: %w", err)
		}
	}

	// SQLite connection pool constraints
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	return db, nil
}

// InitDB loads a SQL schema file into the provided database.
// If the underlying SQLite build does not support FTS5, a minimal
// fallback schema (gospels table + index) is created.
func InitDB(db *sql.DB, schemaPath string) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}
	content, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return err
	}
	if len(content) == 0 {
		return fmt.Errorf("empty schema file: %s", schemaPath)
	}
	if _, err := db.Exec(string(content)); err != nil {
		// Fallback when FTS5 is not available in the SQLite build.
		if strings.Contains(err.Error(), "no such module: fts5") || strings.Contains(err.Error(), "no such function: fts5") {
			createTable := `CREATE TABLE IF NOT EXISTS gospels (
    id INTEGER PRIMARY KEY,
    reference TEXT UNIQUE NOT NULL,
    book TEXT,
    chapter_start INTEGER,
    verse_start INTEGER,
    chapter_end INTEGER,
    verse_end INTEGER,
    text TEXT NOT NULL
);`
			if _, err := db.Exec(createTable); err != nil {
				return fmt.Errorf("fallback create table failed: %w", err)
			}
			createIndex := `CREATE INDEX IF NOT EXISTS idx_gospel_book ON gospels(book);`
			if _, err := db.Exec(createIndex); err != nil {
				return fmt.Errorf("fallback create index failed: %w", err)
			}
			return nil
		}
		return fmt.Errorf("executing schema: %w", err)
	}
	return nil
}
