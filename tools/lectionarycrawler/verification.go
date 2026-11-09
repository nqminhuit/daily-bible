package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type crawlWindow struct {
	from, to                   string
	refresh, acceptCorrections bool
}

var errReferenceConflict = errors.New("source/reference conflict requires calendar review")

func validateWindow(years string, window crawlWindow) (string, error) {
	if (window.from == "") != (window.to == "") {
		return "", fmt.Errorf("-from and -to must be specified together")
	}
	if window.from != "" {
		from, err := time.Parse("2006-01-02", window.from)
		if err != nil {
			return "", fmt.Errorf("invalid -from: %w", err)
		}
		to, err := time.Parse("2006-01-02", window.to)
		if err != nil {
			return "", fmt.Errorf("invalid -to: %w", err)
		}
		if to.Before(from) {
			return "", fmt.Errorf("-to must not precede -from")
		}
		// Derive every year touched by the requested window; no mismatched
		// -years argument can silently omit dates from an explicit window.
		var parts []string
		for year := from.Year(); year <= to.Year(); year++ {
			parts = append(parts, strconv.Itoa(year))
		}
		years = strings.Join(parts, ",")
	}
	if years == "" {
		return "", fmt.Errorf("specify -years or an exact -from/-to window")
	}
	for _, part := range strings.Split(years, ",") {
		year, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || year < 1900 || year > 9999 {
			return "", fmt.Errorf("invalid year %q", part)
		}
	}
	return years, nil
}

// Record exact-date source evidence atomically. A differing existing reference
// requires explicit review: it may be a bad estimate OR a legitimate local Mass.
// Unaccepted conflicts retain the stored ref but invalidate its matching proof.
func storeVerifiedReading(ctx context.Context, db *sql.DB, date, ref, sourceURL string, acceptCorrections bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var previous sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT gospel_ref FROM daily_readings WHERE date = ?`, date).Scan(&previous); err != nil {
		return err
	}
	conflict := previous.Valid && previous.String != "" && previous.String != ref && !acceptCorrections
	if !conflict {
		if _, err := tx.ExecContext(ctx, `UPDATE daily_readings SET gospel_ref = ? WHERE date = ?`, ref, date); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO daily_reading_sources
		(date, gospel_ref, source_url, source_date, verified_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
		gospel_ref = excluded.gospel_ref, source_url = excluded.source_url,
		source_date = excluded.source_date, verified_at = excluded.verified_at,
		celebration = CASE WHEN daily_reading_sources.gospel_ref = excluded.gospel_ref
			THEN daily_reading_sources.celebration ELSE NULL END`,
		date, ref, sourceURL, date, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if conflict {
		return fmt.Errorf("%w: date=%s stored=%q source=%q url=%s; use -accept-corrections only after review", errReferenceConflict, date, previous.String, ref, sourceURL)
	}
	if previous.Valid && previous.String != ref {
		// Keep the mismatch visible in audit logs; the transaction above is
		// authoritative only for the exact date whose page was checked.
		fmt.Printf("CORRECTED date=%s previous=%q source=%q url=%s\n", date, previous.String, ref, sourceURL)
	}
	return nil
}
