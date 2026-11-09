package reading

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/minh/daily-bible/internal/scripture"
)

// HasDateDecision detects explicit exceptions even when their evidence is stale.
// They must not silently fall back to the ordinary/default entry.
func HasDateDecision(ctx context.Context, db *sql.DB, calendar, date, option string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM reading_date_acceptances WHERE calendar_id=? AND date=? AND mass_option=?)`, calendar, date, option).Scan(&exists)
	return exists, err
}

// ReviewDate accepts only this date/calendar/option, never the reusable entry.
// It is useful for a reviewed local exception or a permitted date-specific form.
func ReviewDate(ctx context.Context, db *sql.DB, r EntryReview, celebration string) error {
	ref, err := validateReview(r)
	if err != nil {
		return err
	}
	if r.EvidenceDate != r.Date {
		return fmt.Errorf("date acceptance requires evidence for this exact date")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version string
	if err := tx.QueryRowContext(ctx, `SELECT rules_version FROM reading_calendars WHERE calendar_id=?`, r.CalendarID).Scan(&version); err != nil {
		return err
	}
	var previous, oldURL, oldNote, oldLabel string
	err = tx.QueryRowContext(ctx, `SELECT r.gospel_ref,r.evidence_url,r.review_note,r.celebration FROM reading_date_acceptances a JOIN reading_date_reviews r ON r.id=a.review_id WHERE a.calendar_id=? AND a.date=? AND a.mass_option=? AND r.rules_version=?`, r.CalendarID, r.Date, DayMass, version).Scan(&previous, &oldURL, &oldNote, &oldLabel)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil {
		if !scripture.Equivalent(previous, ref) && !r.Replace {
			return fmt.Errorf("date already accepted as %s; explicit replacement required", previous)
		}
		if scripture.Equivalent(previous, ref) && oldURL == r.EvidenceURL && oldNote == r.Note && oldLabel == celebration {
			return nil
		}
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO reading_date_reviews(calendar_id,rules_version,date,mass_option,gospel_ref,evidence_url,evidence_date,review_note,celebration,reviewed_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, r.CalendarID, version, r.Date, DayMass, ref, r.EvidenceURL, r.EvidenceDate, r.Note, celebration, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO reading_date_acceptances VALUES (?,?,?,?) ON CONFLICT(calendar_id,date,mass_option) DO UPDATE SET review_id=excluded.review_id`, r.CalendarID, r.Date, DayMass, id); err != nil {
		return err
	}
	return tx.Commit()
}
