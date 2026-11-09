package api

import (
	"database/sql"
	"testing"

	dbpkg "github.com/minh/daily-bible/internal/db"
)

// setupTestDB applies the full schema, including daily readings, and uses the
// same single-connection pool as production so :memory: state stays consistent.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := dbpkg.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := dbpkg.ApplyMigrations(db, "../../data/migrations"); err != nil {
		t.Fatal(err)
	}
	return db
}
