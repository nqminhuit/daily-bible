package lectionary

import "testing"

func TestUnsupportedDayHasNoEntryAndCannotNameOne(t *testing.T) {
	d := CalendarDay{Date: "2026-11-15", Observance: "ordinary_time_33_sunday", Temporal: "ordinary_time_33_sunday", Rank: "SUNDAY", SundayCycle: "YEAR_A", WeekdayCycle: "YEAR_2", UnsupportedReason: "national external solemnity unresolved"}
	e, err := d.Entry()
	if err != nil || e.Key != "" {
		t.Fatal("unresolved national day became a temporal fallback", e, err)
	}
	d.GospelEntry = "temporal:ordinary_time_33_sunday:YEAR_A"
	if _, err := d.Entry(); err == nil {
		t.Fatal("unsupported day named a Gospel entry")
	}
}
