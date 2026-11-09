package reading

import (
	"context"
	"database/sql"
	"errors"

	"github.com/minh/daily-bible/internal/scripture"
)

type AuditRow struct {
	Date                 string `json:"date"`
	Status               string `json:"status"`
	LegacyRef            string `json:"legacy_ref,omitempty"`
	ResolvedRef          string `json:"resolved_ref,omitempty"`
	EntryKey             string `json:"entry_key,omitempty"`
	ObservedRef          string `json:"observed_ref,omitempty"`
	EvidenceURL          string `json:"evidence_url,omitempty"`
	EvidenceDate         string `json:"evidence_date,omitempty"`
	ObservationCandidate bool   `json:"observation_candidate,omitempty"`
	ObservationURL       string `json:"observation_url,omitempty"`
	ObservationDate      string `json:"observation_date,omitempty"`
	SourceDifference     bool   `json:"source_difference,omitempty"`
}
type AuditReport struct {
	CalendarID        string         `json:"calendar_id"`
	NetworkRequests   int            `json:"network_requests"`
	Counts            map[string]int `json:"counts"`
	Rows              []AuditRow     `json:"rows"`
	SourceDifferences int            `json:"source_differences"`
}

// Audit classifies local evidence without mutating references or making HTTP
// calls. Legacy disagreement is a review item, not an automatic bulk repair.
func Audit(ctx context.Context, db *sql.DB, calendar, from, to string) (AuditReport, error) {
	report := AuditReport{CalendarID: calendar, Counts: map[string]int{}}
	rows, err := db.QueryContext(ctx, `SELECT date FROM daily_readings WHERE date BETWEEN ? AND ? UNION SELECT date FROM reading_calendar_days WHERE calendar_id=? AND date BETWEEN ? AND ? UNION SELECT date FROM reading_date_acceptances WHERE calendar_id=? AND date BETWEEN ? AND ? ORDER BY date`, from, to, calendar, from, to, calendar, from, to)
	if err != nil {
		return report, err
	}
	var dates []string
	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			rows.Close()
			return report, err
		}
		dates = append(dates, date)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	// Release the rows before nested queries: SQLite has one connection.
	for _, date := range dates {
		r := AuditRow{Date: date}
		if err := db.QueryRowContext(ctx, `SELECT COALESCE(gospel_ref,'') FROM daily_readings WHERE date=?`, date).Scan(&r.LegacyRef); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return report, err
		}
		err := db.QueryRowContext(ctx, `SELECT gospel_ref,source_url,source_date FROM reading_observations WHERE date=? AND calendar_id=? AND mass_option=? ORDER BY id DESC LIMIT 1`, date, SourceCalendar, DayMass).Scan(&r.ObservedRef, &r.ObservationURL, &r.ObservationDate)
		if errors.Is(err, sql.ErrNoRows) {
			err = db.QueryRowContext(ctx, `SELECT gospel_ref,source_url,source_date FROM daily_reading_sources WHERE date=?`, date).Scan(&r.ObservedRef, &r.ObservationURL, &r.ObservationDate)
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return report, err
		}
		m, err := LookupFor(ctx, db, date, calendar, DayMass)
		if errors.Is(err, sql.ErrNoRows) {
			r.Status = "legacy_unverified"
			if calendar != SourceCalendar {
				decided, err := HasDateDecision(ctx, db, calendar, date, DayMass)
				if err != nil {
					return report, err
				}
				if decided {
					r.Status = "date_review_required"
				} else {
					key, version, err := EntryForDate(ctx, db, calendar, date)
					if errors.Is(err, sql.ErrNoRows) {
						r.Status = "unsupported_calendar_date"
					} else if err != nil {
						return report, err
					} else {
						r.Status = "unreviewed_entry"
						r.EntryKey = key
						r.ObservationCandidate, err = HasObservationCandidate(ctx, db, calendar, key, version)
						if err != nil {
							return report, err
						}
					}
				}
			} else if r.ObservedRef != "" && !scripture.Equivalent(r.ObservedRef, r.LegacyRef) {
				r.Status = "conflict"
			}
		} else if err != nil {
			return report, err
		} else {
			r.ResolvedRef = m.Ref
			if r.ObservedRef != "" && !scripture.Equivalent(r.ObservedRef, m.Ref) {
				r.SourceDifference = true
				report.SourceDifferences++
			}
			r.EntryKey = m.EntryKey
			r.EvidenceURL = m.SourceURL
			r.EvidenceDate = m.EvidenceDate
			r.Status = m.ResolutionMethod
			if r.LegacyRef != "" && !scripture.Equivalent(r.LegacyRef, m.Ref) {
				r.Status += "_legacy_difference"
			}
			if _, err := scripture.NormalizeGospelRef(m.Ref); err != nil {
				r.Status = "invalid_reference"
			} else {
				ok, err := scripture.HasText(ctx, db, m.Ref)
				if err != nil {
					return report, err
				}
				if !ok {
					r.Status = "missing_text"
				}
			}
		}
		report.Counts[r.Status]++
		report.Rows = append(report.Rows, r)
	}
	return report, ctx.Err()
}
