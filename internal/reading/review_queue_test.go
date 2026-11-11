package reading

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/minh/daily-bible/internal/lectionary"
)

func TestReviewQueueGroupsAnnualEntriesWithoutApprovingLegacyCandidates(t *testing.T) {
	db := testDB(t)
	m := testManifest(
		testDay("2022-11-07", "ordinary_time_32_monday", "WEEKDAY", "ordinary_time_32_monday", "YEAR_C", "YEAR_2"),
		testDay("2023-11-13", "ordinary_time_32_monday", "WEEKDAY", "ordinary_time_32_monday", "YEAR_A", "YEAR_1"),
		testDay("2026-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	if _, err := db.Exec(`INSERT INTO daily_readings VALUES ('2022-11-07','bad_key','Lc 17,1-6'),('2023-11-13','bad_key','Ga 2,13-22'),('2026-11-09','bad_key','Lc 17,1-6')`); err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 6; n++ {
		if _, err := db.Exec(`INSERT INTO verses(book,chapter,verse,text) VALUES ('Lc',17,?,'text')`, n); err != nil {
			t.Fatal(err)
		}
	}
	if err := ObserveVatican(t.Context(), db, "2022-11-07", "Lc 17,1-6", SourceURL("2022-11-07")); err != nil {
		t.Fatal(err)
	}
	report, err := Audit(t.Context(), db, m.CalendarID, "2022-01-01", "2026-12-31")
	if err != nil || len(report.Rows) != 3 || len(report.Entries) != 2 || report.Counts["unreviewed_entry"] != 3 || report.NetworkRequests != 0 {
		t.Fatalf("queue: %+v %v", report, err)
	}
	var ordinary *EntryAudit
	for i := range report.Entries {
		if len(report.Entries[i].Dates) == 2 {
			ordinary = &report.Entries[i]
		}
	}
	if ordinary == nil || !ordinary.HasReferenceVariants || len(ordinary.LegacyCandidates) != 2 || len(ordinary.Observations) != 1 || len(ordinary.AcceptedEvidence) != 0 {
		t.Fatalf("lost variants/evidence scope: %+v", ordinary)
	}
	if ordinary.LegacyCandidates[0].TextStatus != "missing_text" || ordinary.LegacyCandidates[1].TextStatus != "structurally_available" {
		t.Fatalf("candidate coverage: %+v", ordinary.LegacyCandidates)
	}
	if ordinary.Observations[0].URL != SourceURL("2022-11-07") || ordinary.Observations[0].Date != "2022-11-07" {
		t.Fatal("evidence date fabricated")
	}
	again, err := Audit(t.Context(), db, m.CalendarID, "2022-01-01", "2026-12-31")
	if err != nil || !reflect.DeepEqual(report, again) {
		t.Fatal("nondeterministic queue", err)
	}
	var count int
	for table, want := range map[string]int{"reading_entries": 0, "reading_entry_reviews": 0, "reading_observations": 1} {
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != want {
			t.Fatalf("audit mutated %s: %d %v", table, count, err)
		}
	}
	var ref string
	if err := db.QueryRow(`SELECT gospel_ref FROM daily_readings WHERE date='2023-11-13'`).Scan(&ref); err != nil || ref != "Ga 2,13-22" {
		t.Fatal("legacy ref overwritten", ref, err)
	}
}

func TestAuditUsesObservationTimeRatherThanImportOrder(t *testing.T) {
	db := testDB(t)
	date := "2026-11-09"
	m := testManifest(testDay(date, "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	review(t, db, m.CalendarID, date, "Ga 2,13-22")
	for _, x := range []struct{ ref, stamp string }{{"Lc 9,23-26", "2026-11-09T12:00:00Z"}, {"Ga 2,13-22", "2026-11-09T00:00:00Z"}} {
		if _, err := db.Exec(`INSERT INTO reading_observations(date,calendar_id,mass_option,gospel_ref,source_url,source_date,observed_at) VALUES (?,?,?,?,?,?,?)`, date, SourceCalendar, DayMass, x.ref, SourceURL(date), date, x.stamp); err != nil {
			t.Fatal(err)
		}
	}
	report, err := Audit(t.Context(), db, m.CalendarID, date, date)
	if err != nil || report.Rows[0].ObservedRef != "Lc 9,23-26" || report.SourceDifferences != 1 {
		t.Fatalf("older imported observation hid current fact: %+v %v", report, err)
	}
}

func TestReviewQueueKeepsVersionsSeparateAndFlagsInvalidCandidates(t *testing.T) {
	db := testDB(t)
	report := AuditReport{Rows: []AuditRow{
		{Date: "2026-11-03", EntryKey: "same-entry", RulesVersion: "v1", LegacyRef: "Ga 99,1", Status: "unreviewed_entry"},
		{Date: "2027-11-03", EntryKey: "same-entry", RulesVersion: "v2", LegacyRef: "Lc 14,15-24", Status: "unreviewed_entry"},
		{Date: "2026-11-10", Status: "source_checked"},
	}}
	if err := FinalizeAudit(t.Context(), db, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Entries) != 2 || report.Entries[0].LegacyCandidates[0].TextStatus != "invalid_reference" || report.Entries[1].LegacyCandidates[0].TextStatus != "missing_text" {
		t.Fatalf("version/validation leakage: %+v", report.Entries)
	}
}

func TestManifestRequiresMatchingGospelEntrySchema(t *testing.T) {
	db := testDB(t)
	m := testManifest(testDay("2026-11-03", "ordinary_time_31_tuesday", "WEEKDAY", "ordinary_time_31_tuesday", "YEAR_A", "YEAR_2"))
	for _, version := range []int{0, 1, 2, 3, lectionary.GospelEntrySchemaVersion + 1} {
		m.GospelEntrySchema = version
		if err := ImportCalendar(t.Context(), db, m); err == nil || !strings.Contains(err.Error(), "regenerate the manifest") {
			t.Fatalf("schema %d accepted: %v", version, err)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reading_calendars`).Scan(&count); err != nil || count != 0 {
		t.Fatal("invalid import mutated data", count, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r := AuditReport{}
	if err := FinalizeAudit(ctx, db, &r); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation ignored", err)
	}
}
