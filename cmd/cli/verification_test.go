package main

import (
	"database/sql"
	"strings"
	"testing"
)

func TestCLIRejectsUnverifiedLegacyReading(t *testing.T) {
	path := setupTestDB(t)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM daily_reading_sources`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	out, err := runCLI(t, path, "today")
	if err == nil || !strings.Contains(string(out), "no verified reading") {
		t.Fatalf("legacy prediction treated as verified: %s, %v", out, err)
	}
}
