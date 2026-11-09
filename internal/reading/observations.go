package reading

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/minh/daily-bible/internal/scripture"
)

type sourceExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// ObserveVatican records a page fact, not approval of any national Mass calendar.
// Call only after fetchGospelRef has validated the exact page date.
func ObserveVatican(ctx context.Context, target sourceExecutor, date, ref, sourceURL string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return err
	}
	if sourceURL != SourceURL(date) {
		return fmt.Errorf("observation URL must match its actual date")
	}
	ref, err := scripture.NormalizeGospelRef(ref)
	if err != nil {
		return err
	}
	_, err = target.ExecContext(ctx, `INSERT OR IGNORE INTO reading_observations(date,calendar_id,mass_option,gospel_ref,source_url,source_date,observed_at) VALUES (?,?,?,?,?,?,?)`, date, SourceCalendar, DayMass, ref, sourceURL, date, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// A later conflicting observation of the same provider quarantines the
// source-only selection, not independently reviewed national/local selections.
func sourceObservationAgrees(ctx context.Context, db *sql.DB, date, ref string) (bool, error) {
	exists, err := hasTable(ctx, db, "reading_observations")
	if err != nil || !exists {
		return !exists, err
	}
	var latest string
	err = db.QueryRowContext(ctx, `SELECT gospel_ref FROM reading_observations WHERE date=? AND calendar_id=? AND mass_option=? AND source_url=? ORDER BY julianday(observed_at) DESC,id DESC LIMIT 1`, date, SourceCalendar, DayMass, SourceURL(date)).Scan(&latest)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return scripture.Equivalent(ref, latest), nil
}

// EntryForDate deliberately does not accept the old generated seasonal key.
func EntryForDate(ctx context.Context, db *sql.DB, calendar, date string) (key, version string, err error) {
	err = db.QueryRowContext(ctx, `SELECT d.entry_key,d.rules_version FROM reading_calendar_days d JOIN reading_calendars c ON c.calendar_id=d.calendar_id AND c.rules_version=d.rules_version WHERE d.calendar_id=? AND d.date=? AND d.mass_option=? AND d.entry_key<>''`, calendar, date, DayMass).Scan(&key, &version)
	return
}

func HasObservationCandidate(ctx context.Context, db *sql.DB, calendar, key, version string) (bool, error) {
	var found bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM reading_calendar_days d JOIN reading_observations o ON o.date=d.date WHERE d.calendar_id=? AND d.entry_key=? AND d.rules_version=? AND d.mass_option=? AND o.calendar_id=? AND o.mass_option=?)`, calendar, key, version, DayMass, SourceCalendar, DayMass).Scan(&found)
	return found, err
}
