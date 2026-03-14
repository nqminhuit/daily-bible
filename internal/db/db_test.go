package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// repoRoot finds the repository root by walking up until go.mod is found.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repo root not found")
		}
		dir = parent
	}
}

func TestOpenAndInit(t *testing.T) {
	tmp := filepath.Join(os.TempDir(), "daily_bible_test.db")
	defer os.Remove(tmp)
	// remove in case exists
	os.Remove(tmp)

	dbConn, err := Open(tmp)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer dbConn.Close()

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("could not find repo root: %v", err)
	}
	schemaPath := filepath.Join(root, "data", "schema.sql")
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		t.Fatalf("schema not found at %s", schemaPath)
	}
	if err := InitDB(dbConn, schemaPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	row := dbConn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='gospels'")
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("expected gospels table, scan error: %v", err)
	}
	if name != "gospels" {
		t.Fatalf("unexpected table name: %s", name)
	}
}
