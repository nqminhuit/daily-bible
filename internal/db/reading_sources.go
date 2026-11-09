package db

import (
	"context"
	"database/sql"

	"github.com/minh/daily-bible/data/migrations"
)

// EnsureReadingSources upgrades an existing database without asserting that
// any legacy reading was verified. Safe to call repeatedly before crawling.
func EnsureReadingSources(ctx context.Context, database *sql.DB) error {
	schema, err := migrations.Files.ReadFile("003_reading_sources.sql")
	if err != nil {
		return err
	}
	_, err = database.ExecContext(ctx, string(schema))
	return err
}
