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
	schema, err := migrations.Files.ReadFile("004_reading_cache.sql")
	if err != nil {
		return err
	}
	_, err = database.ExecContext(ctx, string(schema))
	return err
}
