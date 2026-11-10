package reading

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/lectionary"
	"github.com/minh/daily-bible/internal/scripture"
)

const SourceCalendar = "vatican-vi"
const DayMass = "day"

type CalendarManifest struct {
	GospelEntrySchema int                      `json:"gospel_entry_schema"`
	CalendarID        string                   `json:"calendar_id"`
	RulesVersion      string                   `json:"rules_version"`
	SourceURL         string                   `json:"source_url"`
	Days              []lectionary.CalendarDay `json:"days"`
}

func validEvidenceURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil
}

// ImportCalendar imports selected observances, not Gospel mappings. Versions
// invalidate only the resolutions/mappings of this calendar. No legacy refs change.
func ImportCalendar(ctx context.Context, db *sql.DB, m CalendarManifest) error {
	if m.GospelEntrySchema != lectionary.GospelEntrySchemaVersion {
		return fmt.Errorf("calendar Gospel entry schema %d is unsupported; regenerate the manifest with schema %d", m.GospelEntrySchema, lectionary.GospelEntrySchemaVersion)
	}
	if !lectionary.CalendarSupported(m.CalendarID) || !lectionary.ValidRulesVersion(m.RulesVersion) || !validEvidenceURL(m.SourceURL) || len(m.Days) == 0 {
		return fmt.Errorf("invalid calendar manifest")
	}
	if !lectionary.CalendarRulesSupported(m.CalendarID, m.RulesVersion) {
		return fmt.Errorf("unsupported calendar rules: unreviewed Vietnam Romcal profile; provide reviewed national definitions")
	}
	definitions := make([]lectionary.EntryDefinition, len(m.Days))
	payloads := make([]string, len(m.Days))
	seen := map[string]bool{}
	for i, d := range m.Days {
		if seen[d.Date] {
			return fmt.Errorf("duplicate calendar date %s", d.Date)
		}
		seen[d.Date] = true
		e, err := d.Entry()
		if requiresCalendarDetails(m.CalendarID, m.RulesVersion) && !strings.HasPrefix(d.Date, "2026-") {
			return fmt.Errorf("national pilot definitions are limited to reviewed year 2026")
		}
		if err != nil {
			return fmt.Errorf("calendar %s: %w", d.Date, err)
		}
		if d.SourceURL != "" && !validEvidenceURL(d.SourceURL) {
			return fmt.Errorf("calendar %s: invalid day evidence URL", d.Date)
		}
		if d.Mass != nil && m.CalendarID != "vietnam" {
			return fmt.Errorf("calendar %s: assigned national Mass requires Vietnam scope", d.Date)
		}
		if (m.CalendarID == "vietnam" && e.Key != "") || d.Mass != nil {
			if !validEvidenceURL(d.SourceURL) || strings.TrimSpace(d.ReviewNote) == "" {
				return fmt.Errorf("calendar %s: national definition requires its own HTTPS source and review note", d.Date)
			}
		}
		definitions[i] = e
		payload, err := json.Marshal(d)
		if err != nil {
			return err
		}
		payloads[i] = string(payload)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO reading_calendars VALUES (?,?,?,?) ON CONFLICT(calendar_id) DO UPDATE SET rules_version=excluded.rules_version,source_url=excluded.source_url,imported_at=excluded.imported_at WHERE rules_version<>excluded.rules_version OR source_url<>excluded.source_url`, m.CalendarID, m.RulesVersion, m.SourceURL, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	for i, d := range m.Days {
		e := definitions[i]
		// Refuse a different resolution under an unchanged version. The caller
		// must bump its rule/adapter version to invalidate dependent evidence.
		var oldVersion, oldKey, oldObservance, oldLabel, oldDisplaced string
		err := tx.QueryRowContext(ctx, `SELECT rules_version,entry_key,observance,celebration,displaced_entry FROM reading_calendar_days WHERE calendar_id=? AND date=? AND mass_option=?`, m.CalendarID, d.Date, DayMass).Scan(&oldVersion, &oldKey, &oldObservance, &oldLabel, &oldDisplaced)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil && oldVersion == m.RulesVersion && (oldKey != e.Key || oldObservance != e.Observance || oldLabel != e.Celebration || oldDisplaced != e.Displaced) {
			return fmt.Errorf("changed resolution for %s requires a new rules version", d.Date)
		}
		var oldPayload, detailVersion string
		err = tx.QueryRowContext(ctx, `SELECT rules_version,definition_json FROM reading_calendar_day_details WHERE calendar_id=? AND date=? AND mass_option=?`, m.CalendarID, d.Date, DayMass).Scan(&detailVersion, &oldPayload)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil && detailVersion == m.RulesVersion && oldPayload != payloads[i] {
			return fmt.Errorf("changed definition/evidence for %s requires a new rules version", d.Date)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO reading_calendar_days VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(calendar_id,date,mass_option) DO UPDATE SET rules_version=excluded.rules_version,entry_key=excluded.entry_key,observance=excluded.observance,celebration=excluded.celebration,displaced_entry=excluded.displaced_entry WHERE rules_version<>excluded.rules_version`, m.CalendarID, d.Date, DayMass, m.RulesVersion, e.Key, e.Observance, e.Celebration, e.Displaced); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO reading_calendar_day_details VALUES (?,?,?,?,?,?) ON CONFLICT(calendar_id,date,mass_option) DO UPDATE SET rules_version=excluded.rules_version,entry_key=excluded.entry_key,definition_json=excluded.definition_json WHERE rules_version<>excluded.rules_version`, m.CalendarID, d.Date, DayMass, m.RulesVersion, e.Key, payloads[i]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type EntryReview struct {
	CalendarID   string
	Date         string
	Ref          string
	EvidenceURL  string
	EvidenceDate string
	Note         string
	Replace      bool
}

func validateReview(r EntryReview) (string, error) {
	ref, err := scripture.NormalizeGospelRef(r.Ref)
	if err != nil {
		return "", err
	}
	if !lectionary.CalendarSupported(r.CalendarID) || strings.TrimSpace(r.Note) == "" || !validEvidenceURL(r.EvidenceURL) {
		return "", fmt.Errorf("calendar, HTTPS evidence URL and review note required")
	}
	if _, err := time.Parse("2006-01-02", r.Date); err != nil {
		return "", err
	}
	if r.EvidenceDate != "" {
		if _, err := time.Parse("2006-01-02", r.EvidenceDate); err != nil {
			return "", err
		}
	}
	if strings.HasPrefix(r.EvidenceURL, "https://www.vaticannews.va/vi/loi-chua-hang-ngay/") && (r.EvidenceDate == "" || r.EvidenceURL != SourceURL(r.EvidenceDate)) {
		return "", fmt.Errorf("dated evidence must retain its actual source date")
	}
	return ref, nil
}

// ReviewEntry is an explicit operator decision, never an automatic promotion of
// the first/majority legacy ref or of a successful dated crawl.
func ReviewEntry(ctx context.Context, db *sql.DB, r EntryReview) error {
	ref, err := validateReview(r)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version, key string
	if err := tx.QueryRowContext(ctx, `SELECT d.rules_version,d.entry_key FROM reading_calendar_days d JOIN reading_calendars c ON c.calendar_id=d.calendar_id AND c.rules_version=d.rules_version WHERE d.calendar_id=? AND d.date=? AND d.mass_option=? AND d.entry_key<>''`, r.CalendarID, r.Date, DayMass).Scan(&version, &key); err != nil {
		return err
	}
	if !lectionary.CalendarRulesSupported(r.CalendarID, version) {
		return fmt.Errorf("unsupported calendar rules; reusable review requires reviewed calendar definitions")
	}
	if requiresCalendarDetails(r.CalendarID, version) {
		var payload string
		if err := tx.QueryRowContext(ctx, `SELECT definition_json FROM reading_calendar_day_details WHERE calendar_id=? AND date=? AND mass_option=? AND rules_version=? AND entry_key=?`, r.CalendarID, r.Date, DayMass, version, key).Scan(&payload); err != nil {
			return fmt.Errorf("national review requires current calendar definition evidence: %w", err)
		}
		if _, err := decodeCalendarDefinition(payload, key, r.Date, r.CalendarID); err != nil {
			return err
		}
	}
	if expected := lectionary.InvariantGospel(key); expected != "" && !scripture.Equivalent(ref, expected) {
		return fmt.Errorf("reference disagrees with the reviewed proper Gospel %s", expected)
	}
	var previous, previousURL, previousDate, previousNote string
	err = tx.QueryRowContext(ctx, `SELECT r.gospel_ref,r.evidence_url,COALESCE(r.evidence_date,''),r.review_note FROM reading_entries e JOIN reading_entry_reviews r ON r.id=e.review_id WHERE e.calendar_id=? AND e.rules_version=? AND e.entry_key=? AND e.mass_option=?`, r.CalendarID, version, key, DayMass).Scan(&previous, &previousURL, &previousDate, &previousNote)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil {
		if !scripture.Equivalent(previous, ref) && !r.Replace {
			return fmt.Errorf("entry already reviewed as %s; explicit replacement required", previous)
		}
		if scripture.Equivalent(previous, ref) && previousURL == r.EvidenceURL && previousDate == r.EvidenceDate && previousNote == r.Note {
			return nil
		}
	}
	var evidenceDate any
	if r.EvidenceDate != "" {
		evidenceDate = r.EvidenceDate
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO reading_entry_reviews(calendar_id,rules_version,entry_key,mass_option,gospel_ref,evidence_url,evidence_date,review_note,reviewed_at) VALUES (?,?,?,?,?,?,?,?,?)`, r.CalendarID, version, key, DayMass, ref, r.EvidenceURL, evidenceDate, r.Note, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO reading_entries VALUES (?,?,?,?,?) ON CONFLICT(calendar_id,rules_version,entry_key,mass_option) DO UPDATE SET review_id=excluded.review_id`, r.CalendarID, version, key, DayMass, id); err != nil {
		return err
	}
	return tx.Commit()
}
