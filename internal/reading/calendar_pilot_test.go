package reading

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/minh/daily-bible/internal/lectionary"
)

func pilotManifest() CalendarManifest {
	ash := testDay("2026-02-20", "friday_after_ash_wednesday", "WEEKDAY", "friday_after_ash_wednesday", "YEAR_A", "YEAR_2")
	ash.Mass = &lectionary.MassAssignment{Formulary: "ash_wednesday", Celebration: "Lễ Tro"}
	ash.Seasons = []string{"lent"}
	tet := testDay("2026-02-18", "ash_wednesday", "WEEKDAY", "ash_wednesday", "YEAR_A", "YEAR_2")
	tet.Mass = &lectionary.MassAssignment{Formulary: "tet_day_2_ancestors", Celebration: "Mồng Hai Tết"}
	tet.Seasons = []string{"lent"}
	unknown := testDay("2026-11-15", "ordinary_time_33_sunday", "SUNDAY", "ordinary_time_33_sunday", "YEAR_A", "YEAR_2")
	unknown.UnsupportedReason = "Vietnamese Martyrs national default unresolved"
	m := testManifest(tet, ash, unknown)
	m.CalendarID, m.RulesVersion = "vietnam", "vietnam-national-pilot-fixture-v1"
	return m
}

func TestNationalPilotMassProofAndUnsupportedAudit(t *testing.T) {
	db := testDB(t)
	m := pilotManifest()
	importManifest(t, db, m)
	importManifest(t, db, m)
	if _, err := db.Exec(`INSERT INTO daily_readings VALUES ('2026-02-20','legacy','Mt 9,14-15'),('2026-11-15','legacy','Lc 9,23-26')`); err != nil {
		t.Fatal(err)
	}
	for _, day := range m.Days {
		if _, err := LookupFor(t.Context(), db, day.Date, m.CalendarID, DayMass); err != sql.ErrNoRows {
			t.Fatal("definitions approved Gospel", err)
		}
	}
	if err := ReviewEntry(t.Context(), db, EntryReview{CalendarID: m.CalendarID, Date: "2026-02-20", Ref: "Mt 9,14-15", EvidenceURL: "https://example.test/lectionary", Note: "mistaken underlying Friday Gospel"}); err == nil {
		t.Fatal("Friday Gospel approved for named Ash Mass")
	}
	review(t, db, m.CalendarID, "2026-02-20", "Mt 6,1-6.16-18")
	got, err := LookupFor(t.Context(), db, "2026-02-20", m.CalendarID, DayMass)
	if err != nil || got.Ref != "Mt 6,1-6.16-18" || got.Observance != "ash_wednesday" || got.CalendarObservance != "friday_after_ash_wednesday" || got.MassFormulary != "ash_wednesday" || got.Celebration != "Lễ Tro" || got.CalendarSourceURL != m.Days[1].SourceURL || got.Seasons[0] != "lent" {
		t.Fatalf("Mass/calendar metadata lost: %+v %v", got, err)
	}
	if got.EvidenceDate != "" || got.SourceURL != "https://example.test/lectionary" {
		t.Fatal("fabricated dated evidence", got)
	}
	report, err := Audit(t.Context(), db, m.CalendarID, "2026-01-01", "2026-12-31")
	if err != nil || report.Counts["unsupported_calendar_date"] != 1 || report.Counts["unreviewed_entry"] != 1 || report.Counts["missing_text"] != 1 {
		t.Fatal(report.Counts, err)
	}
	for _, row := range report.Rows {
		if row.Date == "2026-11-15" && (!strings.Contains(row.UnsupportedReason, "Martyrs") || row.RulesVersion == "") {
			t.Fatal("ambiguity lost", row)
		}
	}
	var legacy string
	if err := db.QueryRow(`SELECT gospel_ref FROM daily_readings WHERE date='2026-02-20'`).Scan(&legacy); err != nil || legacy != "Mt 9,14-15" {
		t.Fatal("legacy overwritten", legacy, err)
	}
}

func TestNationalPilotMissingProofCannotDriveWarmCacheReviewOrCrawl(t *testing.T) {
	db := testDB(t)
	m := pilotManifest()
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2026-02-20", "Mt 6,1-6.16-18")
	if _, err := db.Exec(`DELETE FROM reading_calendar_day_details WHERE date='2026-02-20'`); err != nil {
		t.Fatal(err)
	}
	if _, err := LookupFor(t.Context(), db, "2026-02-20", m.CalendarID, DayMass); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("proofless warm cache served", err)
	}
	if _, _, err := EntryForDate(t.Context(), db, m.CalendarID, "2026-02-20"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("proofless definition became crawl gap", err)
	}
	if err := ReviewEntry(t.Context(), db, EntryReview{CalendarID: m.CalendarID, Date: "2026-02-20", Ref: "Mt 6,1-6.16-18", EvidenceURL: "https://example.test/gospel", Note: "missing calendar proof"}); err == nil {
		t.Fatal("proofless national mapping approved")
	}
}

func TestNationalDefinitionsRequireOwnProofAndNewVersionForChanges(t *testing.T) {
	db := testDB(t)
	m := pilotManifest()
	m.Days[0].SourceURL = ""
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("national Mass imported without source")
	}
	m = pilotManifest()
	importManifest(t, db, m)
	m.Days[0].ReviewNote = "changed applicability"
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("changed source/decision reused version")
	}
	m.RulesVersion = "vietnam-national-pilot-fixture-v2"
	importManifest(t, db, m)
	m.Days[0].Date = "2027-02-18"
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("2026 pilot extrapolated to another year")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reading_calendar_day_details`).Scan(&count); err != nil || count != 3 {
		t.Fatal("non-idempotent definitions", count, err)
	}
}

func TestCalendarDetailsFailureRollsBackWholeImport(t *testing.T) {
	db := testDB(t)
	if _, err := db.Exec(`CREATE TRIGGER reject_detail BEFORE INSERT ON reading_calendar_day_details BEGIN SELECT RAISE(ABORT,'injected detail failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := ImportCalendar(t.Context(), db, pilotManifest()); err == nil {
		t.Fatal("expected import failure")
	}
	for _, table := range []string{"reading_calendars", "reading_calendar_days", "reading_calendar_day_details"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatal("partial import", table, count, err)
		}
	}
}
