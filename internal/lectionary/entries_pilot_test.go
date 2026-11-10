package lectionary

import "testing"

func TestAssignedMassKeepsUnderlyingCalendarAndSeason(t *testing.T) {
	d := CalendarDay{Date: "2026-02-20", Observance: "friday_after_ash_wednesday", Temporal: "friday_after_ash_wednesday", Rank: "WEEKDAY", SundayCycle: "YEAR_A", WeekdayCycle: "YEAR_2", Seasons: []string{"lent"}}
	base, err := d.Entry()
	if err != nil {
		t.Fatal(err)
	}
	d.Mass = &MassAssignment{Formulary: "ash_wednesday", Celebration: "Lễ Tro"}
	e, err := d.Entry()
	if err != nil || e.Key == base.Key || e.Observance != "ash_wednesday" || e.Celebration != "Lễ Tro" || e.Displaced != d.Observance {
		t.Fatalf("Mass conflated with underlying day: %+v %v", e, err)
	}
	if d.Observance != "friday_after_ash_wednesday" || d.Seasons[0] != "lent" {
		t.Fatal("underlying day/season changed")
	}
	d.WeekdayCycle = "YEAR_1"
	other, err := d.Entry()
	if err != nil || other.Key == e.Key {
		t.Fatal("unreviewed formulary cycle generalization", err)
	}
}

func TestUnsupportedNationalDayCannotBecomeWeekdayOrVigil(t *testing.T) {
	d := CalendarDay{Date: "2026-11-15", Observance: "ordinary_time_33_sunday", Temporal: "ordinary_time_33_sunday", Rank: "SUNDAY", SundayCycle: "YEAR_A", WeekdayCycle: "YEAR_2", UnsupportedReason: "national external solemnity unresolved"}
	e, err := d.Entry()
	if err != nil || e.Key != "" {
		t.Fatal("unresolved national day became a temporal fallback", e, err)
	}
	d.Mass = &MassAssignment{Formulary: "ash_wednesday", Celebration: "Lễ Tro"}
	if _, err := d.Entry(); err == nil {
		t.Fatal("unsupported day assigned a Mass")
	}
	d.UnsupportedReason = ""
	d.Mass.Formulary = "easter_vigil"
	if _, err := d.Entry(); err == nil {
		t.Fatal("unimplemented vigil assigned to default daytime slot")
	}
}
