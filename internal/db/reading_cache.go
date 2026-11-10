package db

import (
	"context"
	"database/sql"

	"github.com/minh/daily-bible/data/migrations"
)

func EnsureReadingCache(ctx context.Context, database *sql.DB) error {
	if err := EnsureReadingSources(ctx, database); err != nil {
		return err
	}
	for _, name := range []string{"004_reading_cache.sql", "005_calendar_day_details.sql"} {
		schema, err := migrations.Files.ReadFile(name)
		if err != nil {
			return err
		}
		if _, err := database.ExecContext(ctx, string(schema)); err != nil {
			return err
		}
	}
	return nil
}
