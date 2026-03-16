package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestOpen_EmptyPath ensures Open returns an error for empty path.
func TestOpen_EmptyPath(t *testing.T) {
	db, err := Open("")
	if err == nil {
		t.Fatalf("expected error for empty path, got nil")
	}
	if db != nil {
		db.Close()
		t.Fatalf("expected nil db on error")
	}
}

// TestOpen_MemorySucceeds verifies Open can open an in-memory SQLite DB and execute queries.
func TestOpen_MemorySucceeds(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) failed: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	var one int
	if err := db.QueryRow("SELECT 1").Scan(&one); err != nil {
		t.Fatalf("simple query failed: %v", err)
	}
	if one != 1 {
		t.Fatalf("unexpected result %d", one)
	}
}

// TestInitDB_NilDB verifies InitDB fails fast when given a nil DB.
func TestInitDB_NilDB(t *testing.T) {
	tmp := t.TempDir()
	schemaPath := filepath.Join(tmp, "schema.sql")
	if err := os.WriteFile(schemaPath, []byte("CREATE TABLE foo(id INTEGER);"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := InitDB(nil, schemaPath); err == nil {
		t.Fatal("expected error when db is nil")
	}
}

// TestInitDB_FileMissingAndEmptyAndSuccess covers missing schema file, empty schema and successful application.
func TestInitDB_FileMissingAndEmptyAndSuccess(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// missing file
	if err := InitDB(db, "/no/such/file.sql"); err == nil {
		t.Fatal("expected error for missing schema file")
	}

	// empty schema file
	tmp := t.TempDir()
	emptyPath := filepath.Join(tmp, "empty.sql")
	if err := os.WriteFile(emptyPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := InitDB(db, emptyPath); err == nil {
		t.Fatal("expected error for empty schema file")
	}

	// success: create table and insert
	schema := "CREATE TABLE test_foo(id INTEGER PRIMARY KEY);\nINSERT INTO test_foo(id) VALUES(42);"
	schemaPath := filepath.Join(tmp, "schema.sql")
	if err := os.WriteFile(schemaPath, []byte(schema), 0644); err != nil {
		t.Fatal(err)
	}
	if err := InitDB(db, schemaPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	var id int
	if err := db.QueryRow("SELECT id FROM test_foo LIMIT 1").Scan(&id); err != nil {
		t.Fatalf("query after InitDB failed: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected id 42 got %d", id)
	}
}
