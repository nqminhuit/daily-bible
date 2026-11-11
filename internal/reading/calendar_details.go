package reading

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/minh/daily-bible/internal/lectionary"
)

// CalendarDefinition reads the exact imported rule evidence, independently of
// Gospel approval. Old snapshots without migration 005 remain readable.
func CalendarDefinition(ctx context.Context, db *sql.DB, calendar, date, version string) (lectionary.CalendarDay, error) {
	var d lectionary.CalendarDay
	exists, err := hasTable(ctx, db, "reading_calendar_day_details")
	if err != nil {
		return d, err
	}
	if !exists {
		return d, sql.ErrNoRows
	}
	var payload, key string
	err = db.QueryRowContext(ctx, `SELECT x.definition_json,x.entry_key FROM reading_calendar_day_details x
	 JOIN reading_calendar_days d ON d.calendar_id=x.calendar_id AND d.date=x.date AND d.mass_option=x.mass_option AND d.rules_version=x.rules_version AND d.entry_key=x.entry_key
	 WHERE x.calendar_id=? AND x.date=? AND x.mass_option=? AND x.rules_version=?`, calendar, date, DayMass, version).Scan(&payload, &key)
	if err != nil {
		return d, err
	}
	return decodeCalendarDefinition(payload, key, date, calendar)
}

func decodeCalendarDefinition(payload, key, date, calendar string) (lectionary.CalendarDay, error) {
	var d lectionary.CalendarDay
	if err := json.Unmarshal([]byte(payload), &d); err != nil {
		return d, fmt.Errorf("invalid calendar definition: %w", err)
	}
	e, err := d.Entry()
	if err != nil || d.Date != date || e.Key != key || (d.SourceURL != "" && !validEvidenceURL(d.SourceURL)) {
		return d, fmt.Errorf("inconsistent calendar definition for %s", date)
	}
	if calendar == "vietnam" && key != "" && (!validEvidenceURL(d.SourceURL) || strings.TrimSpace(d.ReviewNote) == "") {
		return d, fmt.Errorf("national definition lacks source/review note for %s", date)
	}
	return d, nil
}

func requiresCalendarDetails(calendar, version string) bool {
	return calendar == "vietnam"
}

func enrichCalendarMetadata(ctx context.Context, db *sql.DB, m *Metadata) error {
	d, err := CalendarDefinition(ctx, db, m.CalendarID, m.Date, m.RulesVersion)
	if err == sql.ErrNoRows {
		if m.ResolutionMethod == "calendar_resolved" && requiresCalendarDetails(m.CalendarID, m.RulesVersion) {
			return sql.ErrNoRows
		}
		return nil
	}
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// An independently accepted exact-date choice does not derive from
		// this optional definition. Bad auxiliary metadata cannot invalidate it.
		if m.ResolutionMethod == "source_checked" {
			return nil
		}
		return err
	}
	if m.ResolutionMethod == "calendar_resolved" && d.UnsupportedReason != "" {
		return sql.ErrNoRows
	}
	if d.SourceURL != "" {
		m.CalendarSourceURL = d.SourceURL
	}
	m.CalendarReviewNote = d.ReviewNote
	m.CalendarObservance = d.Observance
	m.Temporal = d.Temporal
	m.Seasons = d.Seasons
	if d.Mass != nil {
		m.MassFormulary = d.Mass.Formulary
	}
	return nil
}
