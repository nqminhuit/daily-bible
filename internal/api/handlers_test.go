package api

import (
	"database/sql"
	"testing"

	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/reading"
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

func verifyTestReading(t *testing.T, db *sql.DB, date string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO daily_reading_sources
		(date, gospel_ref, source_url, source_date, verified_at)
		SELECT date, gospel_ref, ?, date, '2026-01-01T00:00:00Z'
		FROM daily_readings WHERE date = ?`, reading.SourceURL(date), date); err != nil {
		t.Fatal(err)
	}
}
