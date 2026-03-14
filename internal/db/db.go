package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
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
// Fails fast if FTS5 is not available: this application requires FTS5.
func InitDB(db *sql.DB, schemaPath string) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}
	if len(content) == 0 {
		return fmt.Errorf("empty schema file: %s", schemaPath)
	}
	res, err := db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	log.Printf("Schema applied, %d rows affected", affected)
	return nil
}
