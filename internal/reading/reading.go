// Package reading provides date-specific, source-verified reading metadata.
// Algorithmic lectionary keys remain calendar estimates, not source evidence.
package reading

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Metadata struct {
	Date          string `json:"date"`
	LectionaryKey string `json:"lectionary_key"`
	Ref           string `json:"ref"`
	SourceURL     string `json:"source_url"`
	VerifiedAt    string `json:"verified_at"`
	Celebration   string `json:"celebration,omitempty"`
}

func SourceURL(date string) string {
	return "https://www.vaticannews.va/vi/loi-chua-hang-ngay/" + strings.ReplaceAll(date, "-", "/") + ".html"
}

// Lookup returns sql.ErrNoRows for predictions, legacy unverified references,
// and stale provenance whose reference no longer matches the daily reading.
func Lookup(ctx context.Context, db *sql.DB, date string) (Metadata, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return Metadata{}, fmt.Errorf("invalid date: %w", err)
	}
	var hasSources bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='daily_reading_sources')`).Scan(&hasSources); err != nil {
		return Metadata{}, err
	}
	if !hasSources {
		return Metadata{}, sql.ErrNoRows
	}
	m := Metadata{Date: date}
	err := db.QueryRowContext(ctx, `
		SELECT dr.lectionary_key, dr.gospel_ref, s.source_url, s.verified_at, COALESCE(s.celebration, '')
		FROM daily_readings dr JOIN daily_reading_sources s ON s.date = dr.date
		WHERE dr.date = ? AND s.gospel_ref = dr.gospel_ref
		  AND s.source_date = dr.date AND s.source_url = ? AND s.verified_at <> ''
	`, date, SourceURL(date)).Scan(&m.LectionaryKey, &m.Ref, &m.SourceURL, &m.VerifiedAt, &m.Celebration)
	return m, err
}
