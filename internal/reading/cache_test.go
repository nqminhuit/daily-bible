package reading

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/lectionary"
	"github.com/minh/daily-bible/internal/scripture"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := dbpkg.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := dbpkg.ApplyMigrations(db, "../../data/migrations"); err != nil {
		t.Fatal(err)
	}
	return db
}
func testDay(date, id, rank, temporal, sc, wc string) lectionary.CalendarDay {
	return lectionary.CalendarDay{Date: date, Observance: id, Rank: rank, Temporal: temporal, SundayCycle: sc, WeekdayCycle: wc, SourceURL: "https://example.test/calendar", ReviewNote: "Reviewed calendar rule fixture"}
}
func testManifest(days ...lectionary.CalendarDay) CalendarManifest {
	return CalendarManifest{GospelEntrySchema: lectionary.GospelEntrySchemaVersion, CalendarID: "roman-general", RulesVersion: "test-v1", SourceURL: "https://example.test/calendar", Days: days}
}
func importManifest(t *testing.T, db *sql.DB, m CalendarManifest) {
	t.Helper()
	if err := ImportCalendar(t.Context(), db, m); err != nil {
		t.Fatal(err)
	}
}
func review(t *testing.T, db *sql.DB, calendar, date, ref string) {
	t.Helper()
	if err := ReviewEntry(t.Context(), db, EntryReview{CalendarID: calendar, Date: date, Ref: ref, EvidenceURL: "https://example.test/lectionary", Note: "Reviewed applicability to this entry and daytime Mass option"}); err != nil {
		t.Fatal(err)
	}
}

func TestProperReferenceReusedAcrossYearsWithoutDatedRows(t *testing.T) {
	db := testDB(t)
	m := testManifest(
		testDay("2025-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_sunday", "YEAR_C", "YEAR_1"),
		testDay("2026-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2"),
		testDay("2027-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_tuesday", "YEAR_B", "YEAR_1"))
	importManifest(t, db, m)
	if _, err := LookupFor(t.Context(), db, "2027-11-09", m.CalendarID, DayMass); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("manifest alone became evidence", err)
	}
	if _, err := db.Exec(`INSERT INTO daily_readings VALUES ('2026-11-09','ordinary_32_mon_II','Lc 17,1-6')`); err != nil {
		t.Fatal(err)
	}
	r := EntryReview{CalendarID: m.CalendarID, Date: "2026-11-09", Ref: "Ga 2,13–22", EvidenceURL: SourceURL("2026-11-09"), EvidenceDate: "2026-11-09", Note: "Reviewed cycle-independent Lateran proper, daytime Mass"}
	if err := ReviewEntry(t.Context(), db, r); err != nil {
		t.Fatal(err)
	}
	for _, date := range []string{"2025-11-09", "2026-11-09", "2027-11-09"} {
		got, err := LookupFor(t.Context(), db, date, m.CalendarID, DayMass)
		if err != nil || got.Ref != "Ga 2,13-22" || got.ResolutionMethod != "calendar_resolved" || got.Celebration == "" {
			t.Fatalf("%s: %+v %v", date, got, err)
		}
		if got.EvidenceDate != "2026-11-09" || got.SourceURL != r.EvidenceURL {
			t.Fatalf("fabricated dated evidence: %+v", got)
		}
	}
	var old string
	if err := db.QueryRow(`SELECT gospel_ref FROM daily_readings`).Scan(&old); err != nil || old != "Lc 17,1-6" {
		t.Fatalf("legacy row mutated: %q %v", old, err)
	}
	if _, err := LookupFor(t.Context(), db, "2026-11-09", SourceCalendar, DayMass); err != sql.ErrNoRows {
		t.Fatalf("cross-calendar fallback: %v", err)
	}
	if _, err := LookupFor(t.Context(), db, "2027-11-09", m.CalendarID, "vigil"); err != sql.ErrNoRows {
		t.Fatalf("Mass option leakage: %v", err)
	}
}

func TestOrdinaryGospelReuseAcrossWeekdayCyclesDoesNotOverrideFeast(t *testing.T) {
	db := testDB(t)
	m := testManifest(
		testDay("2022-11-07", "ordinary_time_32_monday", "WEEKDAY", "ordinary_time_32_monday", "YEAR_C", "YEAR_2"),
		testDay("2028-11-13", "ordinary_time_32_monday", "WEEKDAY", "ordinary_time_32_monday", "YEAR_C", "YEAR_2"),
		testDay("2023-11-13", "ordinary_time_32_monday", "WEEKDAY", "ordinary_time_32_monday", "YEAR_A", "YEAR_1"),
		testDay("2026-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2022-11-07", "Lc 17,1-6")
	for _, date := range []string{"2022-11-07", "2028-11-13", "2023-11-13"} {
		if got, err := LookupFor(t.Context(), db, date, m.CalendarID, DayMass); err != nil || got.Ref != "Lc 17,1-6" {
			t.Fatalf("ordinary reuse: %+v %v", got, err)
		}
	}
	for _, date := range []string{"2026-11-09"} {
		if _, err := LookupFor(t.Context(), db, date, m.CalendarID, DayMass); err != sql.ErrNoRows {
			t.Fatalf("wrong entry reused on %s: %v", date, err)
		}
	}
}

func TestVersionInvalidationIsScopedAndHistoryPreserved(t *testing.T) {
	db := testDB(t)
	m := testManifest(testDay("2026-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2026-11-09", "Ga 2,13-22")
	v := m
	v.CalendarID = "vietnam"
	importManifest(t, db, v)
	review(t, db, v.CalendarID, "2026-11-09", "Ga 2,13-22")
	m.RulesVersion = "test-v2"
	importManifest(t, db, m)
	if _, err := LookupFor(t.Context(), db, "2026-11-09", m.CalendarID, DayMass); err != sql.ErrNoRows {
		t.Fatalf("stale dependency accepted: %v", err)
	}
	if _, err := LookupFor(t.Context(), db, "2026-11-09", v.CalendarID, DayMass); err != nil {
		t.Fatalf("unrelated calendar invalidated: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reading_entry_reviews`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("review history lost: %d %v", count, err)
	}
}

func TestExplicitReviewValidationAndReplacement(t *testing.T) {
	db := testDB(t)
	m := testManifest(testDay("2026-11-03", "ordinary_time_31_tuesday", "WEEKDAY", "ordinary_time_31_tuesday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	r := EntryReview{CalendarID: m.CalendarID, Date: "2026-11-03", Ref: "Lc 14,15-24", EvidenceURL: SourceURL("2026-11-03"), EvidenceDate: "2026-11-03", Note: "Reviewed standard weekday Gospel and default form"}
	for _, bad := range []EntryReview{
		{CalendarID: r.CalendarID, Date: r.Date, Ref: "Ga 999,1", EvidenceURL: r.EvidenceURL, Note: r.Note},
		{CalendarID: r.CalendarID, Date: r.Date, Ref: r.Ref, EvidenceURL: r.EvidenceURL, EvidenceDate: "2026-11-04", Note: r.Note},
		{CalendarID: r.CalendarID, Date: r.Date, Ref: r.Ref, EvidenceURL: "http://example.test", Note: r.Note},
		{CalendarID: r.CalendarID, Date: r.Date, Ref: r.Ref, EvidenceURL: r.EvidenceURL, EvidenceDate: r.EvidenceDate},
	} {
		if err := ReviewEntry(t.Context(), db, bad); err == nil {
			t.Fatalf("invalid review accepted: %+v", bad)
		}
	}
	if err := ReviewEntry(t.Context(), db, r); err != nil {
		t.Fatal(err)
	}
	if err := ReviewEntry(t.Context(), db, r); err != nil {
		t.Fatal(err)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM reading_entry_reviews`).Scan(&count)
	if count != 1 {
		t.Fatalf("identical review is not idempotent: %d", count)
	}
	r.Ref = "Lc 14,15-23"
	if err := ReviewEntry(t.Context(), db, r); err == nil {
		t.Fatal("mapping replaced without review flag")
	}
	r.Replace = true
	if err := ReviewEntry(t.Context(), db, r); err != nil {
		t.Fatal(err)
	}
	db.QueryRow(`SELECT COUNT(*) FROM reading_entry_reviews`).Scan(&count)
	if count != 2 {
		t.Fatalf("replacement erased review history: %d", count)
	}
}

func TestManifestValidationAndAtomicImport(t *testing.T) {
	db := testDB(t)
	day := testDay("2026-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2")
	m := testManifest(day)
	importManifest(t, db, m)
	importManifest(t, db, m)
	duplicate := testManifest(day, day)
	if err := ImportCalendar(t.Context(), db, duplicate); err == nil {
		t.Fatal("duplicate date accepted")
	}
	m.Days[0].Observance = "ordinary_time_32_monday"
	m.Days[0].Rank = "WEEKDAY"
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("changed resolution reused the same version")
	}
	m.RulesVersion = "changed"
	if _, err := db.Exec(`CREATE TRIGGER reject_day BEFORE UPDATE ON reading_calendar_days BEGIN SELECT RAISE(ABORT,'injected failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("expected rollback")
	}
	var version string
	db.QueryRow(`SELECT rules_version FROM reading_calendars`).Scan(&version)
	if version != "test-v1" {
		t.Fatalf("partial import changed calendar version: %s", version)
	}
}

func TestUnreviewedVietnamProviderIsBlockedWithoutLosingDateEvidence(t *testing.T) {
	db := testDB(t)
	m := testManifest(testDay("2026-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2"))
	m.CalendarID = "vietnam"
	m.RulesVersion = "romcal-3.0.0-dev.144-unreviewed"
	if err := ImportCalendar(t.Context(), db, m); err == nil {
		t.Fatal("unreviewed national provider imported")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reading_calendars`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rejected import mutated data: %d %v", count, err)
	}
	// Simulate a cache made by the previous adapter before this discrepancy
	// was identified. Readers and maintenance must also fail closed for it.
	m.RulesVersion = "reviewed-fixture-v1"
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2026-11-09", "Ga 2,13-22")
	for _, table := range []string{"reading_calendars", "reading_calendar_days", "reading_entries", "reading_entry_reviews"} {
		if _, err := db.Exec(`UPDATE ` + table + ` SET rules_version='romcal-3.0.0-dev.144-unreviewed'`); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LookupFor(t.Context(), db, "2026-11-09", m.CalendarID, DayMass); err != sql.ErrNoRows {
		t.Fatalf("unsafe warm cache used: %v", err)
	}
	if _, _, err := EntryForDate(t.Context(), db, m.CalendarID, "2026-11-09"); err != sql.ErrNoRows {
		t.Fatalf("unsafe crawl candidate: %v", err)
	}
	r := EntryReview{CalendarID: m.CalendarID, Date: "2026-11-09", Ref: "Ga 2,13-22", EvidenceURL: SourceURL("2026-11-09"), EvidenceDate: "2026-11-09", Note: "Independently reviewed scoped date"}
	if err := ReviewEntry(t.Context(), db, r); err == nil {
		t.Fatal("reusable mapping approved against unsupported calendar rules")
	}
	report, err := Audit(t.Context(), db, m.CalendarID, "2026-11-09", "2026-11-09")
	if err != nil || report.Counts["unsupported_calendar_rules"] != 1 {
		t.Fatalf("unsupported policy not visible: %+v %v", report, err)
	}
	if err := ReviewDate(t.Context(), db, r, "Reviewed date label"); err != nil {
		t.Fatal(err)
	}
	got, err := LookupFor(t.Context(), db, r.Date, m.CalendarID, DayMass)
	if err != nil || got.ResolutionMethod != "source_checked" || got.Ref != r.Ref {
		t.Fatalf("independent date evidence was lost: %+v %v", got, err)
	}
}

func TestOtherScopeObservationDoesNotInvalidateReviewedMapping(t *testing.T) {
	db := testDB(t)
	m := testManifest(testDay("2026-11-15", "ordinary_time_33_sunday", "SUNDAY", "ordinary_time_33_sunday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2026-11-15", "Mt 25,14-30")
	// A synthetic differing source choice is not evidence that this separately
	// reviewed calendar/option is wrong. No claim about a real parish is made.
	if err := ObserveVatican(t.Context(), db, "2026-11-15", "Lc 9,23-26", SourceURL("2026-11-15")); err != nil {
		t.Fatal(err)
	}
	if _, err := LookupFor(t.Context(), db, "2026-11-15", m.CalendarID, DayMass); err != nil {
		t.Fatal(err)
	}
	if _, err := LookupFor(t.Context(), db, "2026-11-15", "vietnam", DayMass); err != sql.ErrNoRows {
		t.Fatal("unapproved calendar choice accepted", err)
	}
}

func TestAuditPreservesLegacyRowsAndDetectsTextGaps(t *testing.T) {
	db := testDB(t)
	m := testManifest(testDay("2026-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2026-11-09", "Ga 2,13-22")
	if _, err := db.Exec(`INSERT INTO daily_readings VALUES ('2026-11-09','estimated','Lc 17,1-6'); INSERT INTO verses(book,chapter,verse,text) VALUES ('Ga',2,13,'first verse only')`); err != nil {
		t.Fatal(err)
	}
	report, err := Audit(t.Context(), db, m.CalendarID, "2026-11-09", "2026-11-09")
	if err != nil || report.NetworkRequests != 0 || report.Counts["missing_text"] != 1 {
		t.Fatalf("audit: %+v %v", report, err)
	}
	for n := 14; n <= 22; n++ {
		if _, err := db.Exec(`INSERT INTO verses(book,chapter,verse,text) VALUES ('Ga',2,?,'text')`, n); err != nil {
			t.Fatal(err)
		}
	}
	report, err = Audit(t.Context(), db, m.CalendarID, "2026-11-09", "2026-11-09")
	if err != nil || report.Counts["calendar_resolved_legacy_difference"] != 1 {
		t.Fatalf("reuse audit: %+v %v", report, err)
	}
	if !strings.Contains(report.Rows[0].LegacyRef, "Lc") {
		t.Fatal("legacy evidence lost")
	}
	if ok, err := scripture.HasText(t.Context(), db, "Ga 2,13-22"); err != nil || !ok {
		t.Fatal(ok, err)
	}
}
