package reading

import "testing"

func TestIndependentDateDecisionSurvivesCorruptAuxiliaryDefinition(t *testing.T) {
	db := testDB(t)
	m := vietnamManifest()
	importManifest(t, db, m)
	r := EntryReview{CalendarID: m.CalendarID, Date: martyrsDate, Ref: "Lc 9,23-26", EvidenceURL: "https://example.test/exact-date", EvidenceDate: martyrsDate, Note: "Independently reviewed exact date, not derived from calendar metadata"}
	if err := ReviewDate(t.Context(), db, r, "Exact-date Martyrs"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE reading_calendar_day_details SET definition_json='broken' WHERE date='2026-11-24'`); err != nil {
		t.Fatal(err)
	}
	got, err := LookupFor(t.Context(), db, r.Date, m.CalendarID, DayMass)
	if err != nil || got.Ref != r.Ref || got.ResolutionMethod != "source_checked" || got.Celebration != "Exact-date Martyrs" {
		t.Fatal("auxiliary metadata invalidated independent evidence", got, err)
	}
	report, err := Audit(t.Context(), db, m.CalendarID, r.Date, r.Date)
	if err != nil || len(report.Rows) != 1 || report.Rows[0].ResolvedRef != r.Ref || report.Rows[0].CalendarDefinitionError == "" || report.Rows[0].Status != "missing_text" {
		t.Fatal("audit lost independent evidence or hid corrupt auxiliary data", report, err)
	}
}
