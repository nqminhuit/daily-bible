// Package reading resolves dates using direct source evidence or a reviewed
// reusable entry in an explicitly selected, versioned calendar.
package reading

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/lectionary"
)

type Metadata struct {
	Date              string `json:"date"`
	LectionaryKey     string `json:"lectionary_key"`
	Ref               string `json:"ref"`
	SourceURL         string `json:"source_url"`
	VerifiedAt        string `json:"verified_at"`
	Celebration       string `json:"celebration"`
	ResolutionMethod  string `json:"resolution_method"`
	CalendarID        string `json:"calendar_id"`
	MassOption        string `json:"mass_option"`
	EntryKey          string `json:"entry_key,omitempty"`
	RulesVersion      string `json:"rules_version,omitempty"`
	EvidenceDate      string `json:"evidence_date,omitempty"`
	ReviewNote        string `json:"review_note,omitempty"`
	CalendarSourceURL string `json:"calendar_source_url,omitempty"`
	Observance        string `json:"observance,omitempty"`
}

func SourceURL(date string) string {
	return "https://www.vaticannews.va/vi/loi-chua-hang-ngay/" + strings.ReplaceAll(date, "-", "/") + ".html"
}

// DefaultCalendar preserves the source-only policy until a local calendar is
// explicitly selected. Importing a profile never silently activates it.
func DefaultCalendar() string {
	if id := os.Getenv("READING_CALENDAR"); id != "" {
		return id
	}
	return SourceCalendar
}
func DefaultMassOption() string {
	if option := os.Getenv("READING_MASS_OPTION"); option != "" {
		return option
	}
	return DayMass
}
func Lookup(ctx context.Context, db *sql.DB, date string) (Metadata, error) {
	return LookupFor(ctx, db, date, DefaultCalendar(), DefaultMassOption())
}
func hasTable(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`, table).Scan(&exists)
	return exists, err
}

// LookupFor never falls back between calendars/options or to legacy predictions.
// It only reads local evidence; no HTTP is needed, including for future dates.
func LookupFor(ctx context.Context, db *sql.DB, date, calendar, option string) (Metadata, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return Metadata{}, fmt.Errorf("invalid date: %w", err)
	}
	if option != DayMass {
		return Metadata{}, sql.ErrNoRows
	}
	m := Metadata{Date: date, CalendarID: calendar, MassOption: option}
	if calendar == SourceCalendar {
		exists, err := hasTable(ctx, db, "daily_reading_sources")
		if err != nil {
			return m, err
		}
		if !exists {
			return m, sql.ErrNoRows
		}
		err = db.QueryRowContext(ctx, `SELECT dr.lectionary_key,dr.gospel_ref,s.source_url,s.verified_at,COALESCE(s.celebration,'')
		FROM daily_readings dr JOIN daily_reading_sources s ON s.date=dr.date
		WHERE dr.date=? AND s.gospel_ref=dr.gospel_ref AND s.source_date=dr.date AND s.source_url=? AND s.verified_at<>''`, date, SourceURL(date)).Scan(&m.LectionaryKey, &m.Ref, &m.SourceURL, &m.VerifiedAt, &m.Celebration)
		m.ResolutionMethod = "source_checked"
		m.EvidenceDate = date
		if err == nil {
			agrees, checkErr := sourceObservationAgrees(ctx, db, date, m.Ref)
			if checkErr != nil {
				return m, checkErr
			}
			if !agrees {
				return m, sql.ErrNoRows
			}
		}
		return m, err
	}
	exists, err := hasTable(ctx, db, "reading_calendars")
	if err != nil {
		return m, err
	}
	if !exists {
		return m, sql.ErrNoRows
	}
	err = db.QueryRowContext(ctx, `SELECT COALESCE(dr.lectionary_key,''),r.gospel_ref,r.evidence_url,r.reviewed_at,r.celebration,r.rules_version,r.evidence_date,r.review_note,c.source_url
	 FROM reading_date_acceptances a
	 JOIN reading_date_reviews r ON r.id=a.review_id AND r.calendar_id=a.calendar_id AND r.date=a.date AND r.mass_option=a.mass_option AND r.evidence_date=a.date
	 JOIN reading_calendars c ON c.calendar_id=r.calendar_id AND c.rules_version=r.rules_version
	 LEFT JOIN daily_readings dr ON dr.date=a.date
	 WHERE a.calendar_id=? AND a.date=? AND a.mass_option=? AND r.review_note<>'' AND r.evidence_url<>'' AND r.reviewed_at<>''`, calendar, date, option).Scan(&m.LectionaryKey, &m.Ref, &m.SourceURL, &m.VerifiedAt, &m.Celebration, &m.RulesVersion, &m.EvidenceDate, &m.ReviewNote, &m.CalendarSourceURL)
	if err == nil {
		m.ResolutionMethod = "source_checked"
		return m, nil
	}
	if err != sql.ErrNoRows {
		return m, err
	}
	if decided, err := HasDateDecision(ctx, db, calendar, date, option); err != nil {
		return m, err
	} else if decided {
		return m, sql.ErrNoRows
	}
	err = db.QueryRowContext(ctx, `SELECT COALESCE(dr.lectionary_key,''),r.gospel_ref,r.evidence_url,r.reviewed_at,
	 d.celebration,d.entry_key,d.rules_version,COALESCE(r.evidence_date,''),r.review_note,c.source_url,d.observance
	 FROM reading_calendar_days d
	 JOIN reading_calendars c ON c.calendar_id=d.calendar_id AND c.rules_version=d.rules_version
	 JOIN reading_entries e ON e.calendar_id=d.calendar_id AND e.rules_version=d.rules_version AND e.entry_key=d.entry_key AND e.mass_option=d.mass_option
	 JOIN reading_entry_reviews r ON r.id=e.review_id AND r.calendar_id=e.calendar_id AND r.rules_version=e.rules_version AND r.entry_key=e.entry_key AND r.mass_option=e.mass_option
	 LEFT JOIN daily_readings dr ON dr.date=d.date
	 WHERE d.calendar_id=? AND d.date=? AND d.mass_option=? AND d.entry_key<>'' AND r.review_note<>'' AND r.evidence_url<>'' AND r.reviewed_at<>''`, calendar, date, option).Scan(&m.LectionaryKey, &m.Ref, &m.SourceURL, &m.VerifiedAt, &m.Celebration, &m.EntryKey, &m.RulesVersion, &m.EvidenceDate, &m.ReviewNote, &m.CalendarSourceURL, &m.Observance)
	m.ResolutionMethod = "calendar_resolved"
	if err == nil && !lectionary.CalendarRulesSupported(calendar, m.RulesVersion) {
		return m, sql.ErrNoRows
	}
	return m, err
}

// Response retains existing fields while making the evidence basis explicit.
// verified means supported resolution, not necessarily a dated page crawl.
func (m Metadata) Response(verses string) map[string]any {
	return map[string]any{"date": m.Date, "lectionary_key": m.LectionaryKey, "ref": m.Ref, "verses": verses,
		"verified": true, "lectionary_key_is_estimate": true, "source_url": m.SourceURL, "verified_at": m.VerifiedAt, "celebration": m.Celebration,
		"resolution_method": m.ResolutionMethod, "calendar_id": m.CalendarID, "mass_option": m.MassOption,
		"entry_key": m.EntryKey, "rules_version": m.RulesVersion, "evidence_date": m.EvidenceDate,
		"calendar_source_url": m.CalendarSourceURL, "observance": m.Observance}
}
