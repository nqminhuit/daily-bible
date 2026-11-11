package reading

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/minh/daily-bible/internal/lectionary"
)

const (
	martyrsDate  = "2026-11-24"
	martyrsEntry = "observance:andrew_dung_lac_priest_and_companions_martyrs"
)

func vietnamManifest() CalendarManifest {
	martyrs := testDay(martyrsDate, "andrew_dung_lac_priest_and_companions_martyrs", "SOLEMNITY", "ordinary_time_34_tuesday", "YEAR_A", "YEAR_2")
	martyrs.Celebration = "Các Thánh Tử Đạo Việt Nam"
	martyrs.Seasons = []string{"ordinary_time"}
	ash := testDay("2026-02-18", "ash_wednesday", "WEEKDAY", "ash_wednesday", "YEAR_A", "YEAR_2")
	ash.Seasons = []string{"lent"}
	holy := testDay("2027-03-25", "holy_thursday", "TRIDUUM", "holy_thursday", "YEAR_B", "YEAR_1")
	holy.GospelEntry, holy.SourceURL, holy.ReviewNote = "", "", ""
	holy.UnsupportedReason = "no daytime Gospel entry is defined for this day"
	m := testManifest(ash, martyrs, holy)
	m.CalendarID, m.RulesVersion = "vietnam", "vietnam-general-fixture-v1"
	return m
}

func TestVietnamDefinitionsAndUnsupportedAudit(t *testing.T) {
	db := testDB(t)
	m := vietnamManifest()
	importManifest(t, db, m)
	importManifest(t, db, m)
	if _, err := db.Exec(`INSERT INTO daily_readings VALUES ('2026-11-24','legacy','Mt 9,14-15')`); err != nil {
		t.Fatal(err)
	}
	for _, day := range m.Days {
		if _, err := LookupFor(t.Context(), db, day.Date, m.CalendarID, DayMass); err != sql.ErrNoRows {
			t.Fatal("definitions approved Gospel", err)
		}
	}
	review(t, db, m.CalendarID, martyrsDate, "Lc 9,23-26")
	got, err := LookupFor(t.Context(), db, martyrsDate, m.CalendarID, DayMass)
	if err != nil || got.Ref != "Lc 9,23-26" || got.Observance != "andrew_dung_lac_priest_and_companions_martyrs" || got.CalendarObservance != got.Observance || got.Temporal != "ordinary_time_34_tuesday" || got.Celebration != "Các Thánh Tử Đạo Việt Nam" || got.CalendarSourceURL != m.Days[1].SourceURL || got.Seasons[0] != "ordinary_time" {
		t.Fatalf("calendar metadata lost: %+v %v", got, err)
	}
	if got.EvidenceDate != "" || got.SourceURL != "https://example.test/lectionary" {
		t.Fatal("fabricated dated evidence", got)
	}
	report, err := Audit(t.Context(), db, m.CalendarID, "2026-01-01", "2027-12-31")
	if err != nil || report.Counts["unsupported_calendar_date"] != 1 || report.Counts["unreviewed_entry"] != 1 || report.Counts["missing_text"] != 1 {
		t.Fatal(report.Counts, err)
	}
	for _, row := range report.Rows {
		if row.Date == "2027-03-25" && (!strings.Contains(row.UnsupportedReason, "daytime") || row.RulesVersion == "") {
			t.Fatal("unsupported reason lost", row)
		}
	}
	var legacy string
	if err := db.QueryRow(`SELECT gospel_ref FROM daily_readings WHERE date='2026-11-24'`).Scan(&legacy); err != nil || legacy != "Mt 9,14-15" {
		t.Fatal("legacy overwritten", legacy, err)
	}
}

func TestVietnamMissingProofCannotDriveWarmCacheReviewOrCrawl(t *testing.T) {
	db := testDB(t)
	m := vietnamManifest()
	importManifest(t, db, m)
	review(t, db, m.CalendarID, martyrsDate, "Lc 9,23-26")
	if _, err := db.Exec(`DELETE FROM reading_calendar_day_details WHERE date=?`, martyrsDate); err != nil {
		t.Fatal(err)
	}
	if _, err := LookupFor(t.Context(), db, martyrsDate, m.CalendarID, DayMass); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("proofless warm cache served", err)
	}
	if _, _, err := EntryForDate(t.Context(), db, m.CalendarID, martyrsDate); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("proofless definition became crawl gap", err)
	}
	if err := ReviewEntry(t.Context(), db, EntryReview{CalendarID: m.CalendarID, Date: martyrsDate, Ref: "Lc 9,23-26", EvidenceURL: "https://example.test/gospel", Note: "missing calendar proof"}); err == nil {
		t.Fatal("proofless national mapping approved")
	}
}

func TestVietnamDefinitionsRequireOwnProofAndNewVersionForChanges(t *testing.T) {
	db := testDB(t)
	m := vietnamManifest()
	m.Days[1].SourceURL = ""
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("national day imported without source")
	}
	m = vietnamManifest()
	importManifest(t, db, m)
	m.Days[1].ReviewNote = "changed applicability"
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("changed source/decision reused version")
	}
	m.RulesVersion = "vietnam-general-fixture-v2"
	importManifest(t, db, m)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reading_calendar_day_details`).Scan(&count); err != nil || count != 3 {
		t.Fatal("non-idempotent definitions", count, err)
	}
}

func TestVietnamManifestImportsAnyYearAndSharesEntriesAcrossYears(t *testing.T) {
	db := testDB(t)
	m := vietnamManifest()
	next := testDay("2027-11-24", "andrew_dung_lac_priest_and_companions_martyrs", "SOLEMNITY", "ordinary_time_34_wednesday", "YEAR_B", "YEAR_1")
	m.Days = append(m.Days, next)
	importManifest(t, db, m)
	review(t, db, m.CalendarID, martyrsDate, "Lc 9,23-26")
	got, err := LookupFor(t.Context(), db, "2027-11-24", m.CalendarID, DayMass)
	if err != nil || got.Ref != "Lc 9,23-26" || got.ResolutionMethod != "calendar_resolved" {
		t.Fatalf("2027 date did not reuse the entry: %+v %v", got, err)
	}
}

func TestCalendarDetailsFailureRollsBackWholeImport(t *testing.T) {
	db := testDB(t)
	if _, err := db.Exec(`CREATE TRIGGER reject_detail BEFORE INSERT ON reading_calendar_day_details BEGIN SELECT RAISE(ABORT,'injected detail failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := ImportCalendar(t.Context(), db, vietnamManifest()); err == nil {
		t.Fatal("expected import failure")
	}
	for _, table := range []string{"reading_calendars", "reading_calendar_days", "reading_calendar_day_details"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatal("partial import", table, count, err)
		}
	}
}

func TestDefinitionsWithAssignedMassOrMissingEntryAreRejected(t *testing.T) {
	db := testDB(t)
	m := vietnamManifest()
	m.Days[1].Mass = &lectionary.MassAssignment{Formulary: "ash_wednesday", Celebration: "Lễ Tro"}
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("assigned Mass imported")
	}
	m = vietnamManifest()
	m.Days[1].GospelEntry = ""
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("supported day without Gospel entry imported")
	}
}
