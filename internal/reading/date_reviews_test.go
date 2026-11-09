package reading

import (
	"database/sql"
	"testing"
)

func TestDateReviewDoesNotPromoteOrReplaceReusableEntry(t *testing.T) {
	db := testDB(t)
	m := testManifest(
		testDay("2022-11-07", "ordinary_time_32_monday", "WEEKDAY", "ordinary_time_32_monday", "YEAR_C", "YEAR_2"),
		testDay("2028-11-13", "ordinary_time_32_monday", "WEEKDAY", "ordinary_time_32_monday", "YEAR_C", "YEAR_2"))
	m.CalendarID = "vietnam"
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2022-11-07", "Lc 17,1-6")
	// Synthetic reviewed local exception: this test asserts scoping, not an
	// actual celebration or permitted reference for this historical date.
	r := EntryReview{CalendarID: m.CalendarID, Date: "2022-11-07", Ref: "Lc 9,23-26", EvidenceURL: "https://example.test/local/2022-11-07", EvidenceDate: "2022-11-07", Note: "Reviewed local date exception, not a reusable ordinary entry"}
	if err := ReviewDate(t.Context(), db, r, "Reviewed local observance"); err != nil {
		t.Fatal(err)
	}
	if err := ReviewDate(t.Context(), db, r, "Reviewed local observance"); err != nil {
		t.Fatal(err)
	}
	got, err := LookupFor(t.Context(), db, r.Date, m.CalendarID, DayMass)
	if err != nil || got.Ref != r.Ref || got.ResolutionMethod != "source_checked" || got.Celebration != "Reviewed local observance" {
		t.Fatalf("date selection: %+v %v", got, err)
	}
	got, err = LookupFor(t.Context(), db, "2028-11-13", m.CalendarID, DayMass)
	if err != nil || got.Ref != "Lc 17,1-6" || got.ResolutionMethod != "calendar_resolved" {
		t.Fatalf("exception propagated: %+v %v", got, err)
	}
	r.Ref = "Lc 9,23-25"
	if err := ReviewDate(t.Context(), db, r, ""); err == nil {
		t.Fatal("replacement accepted without explicit review flag")
	}
	r.Replace = true
	if err := ReviewDate(t.Context(), db, r, ""); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reading_date_reviews`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("history/idempotence: %d %v", count, err)
	}
}

func TestStaleDateExceptionDoesNotFallBackToOrdinaryEntry(t *testing.T) {
	db := testDB(t)
	m := testManifest(testDay("2026-11-03", "ordinary_time_31_tuesday", "WEEKDAY", "ordinary_time_31_tuesday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2026-11-03", "Lc 14,15-24")
	r := EntryReview{CalendarID: m.CalendarID, Date: "2026-11-03", Ref: "Lc 9,23-26", EvidenceURL: "https://example.test/date", EvidenceDate: "2026-11-03", Note: "Synthetic reviewed date choice for scoping regression"}
	if err := ReviewDate(t.Context(), db, r, ""); err != nil {
		t.Fatal(err)
	}
	m.RulesVersion = "test-v2"
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2026-11-03", "Lc 14,15-24")
	if _, err := LookupFor(t.Context(), db, r.Date, m.CalendarID, DayMass); err != sql.ErrNoRows {
		t.Fatalf("stale exception fell back: %v", err)
	}
	report, err := Audit(t.Context(), db, m.CalendarID, r.Date, r.Date)
	if err != nil || report.Counts["date_review_required"] != 1 {
		t.Fatalf("stale decision diagnosis: %+v %v", report, err)
	}
	r.EvidenceDate = "2026-11-04"
	if err := ReviewDate(t.Context(), db, r, ""); err == nil {
		t.Fatal("wrong-date evidence accepted")
	}
}

func TestLatestSameSourceConflictQuarantinesOnlySourceCalendar(t *testing.T) {
	db := testDB(t)
	date := "2026-11-09"
	if _, err := db.Exec(`INSERT INTO daily_readings VALUES (?, 'estimate', 'Lc 17,1-6'); INSERT INTO daily_reading_sources VALUES (?, 'Lc 17,1-6', ?, ?, '2026-11-09T00:00:00Z', NULL)`, date, date, SourceURL(date), date); err != nil {
		t.Fatal(err)
	}
	if _, err := LookupFor(t.Context(), db, date, SourceCalendar, DayMass); err != nil {
		t.Fatal(err)
	}
	if err := ObserveVatican(t.Context(), db, date, "Ga 2,13-22", SourceURL(date)); err != nil {
		t.Fatal(err)
	}
	if _, err := LookupFor(t.Context(), db, date, SourceCalendar, DayMass); err != sql.ErrNoRows {
		t.Fatalf("known source disagreement served: %v", err)
	}
	m := testManifest(testDay(date, "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	review(t, db, m.CalendarID, date, "Ga 2,13-22")
	if _, err := LookupFor(t.Context(), db, date, m.CalendarID, DayMass); err != nil {
		t.Fatalf("independent calendar poisoned: %v", err)
	}
}
