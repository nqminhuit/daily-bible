package lectionary

import (
	"strings"
	"testing"
)

func weekdayDay(date, cycle string) CalendarDay {
	return CalendarDay{Date: date, Observance: "ordinary_time_31_tuesday", Temporal: "ordinary_time_31_tuesday", Rank: "WEEKDAY", SundayCycle: cycle, WeekdayCycle: "YEAR_2", GospelEntry: "temporal:ordinary_time_31_tuesday"}
}

func TestGospelIdentityUsesAnnualWeekdaysButThreeYearSundays(t *testing.T) {
	first, err := weekdayDay("2026-11-03", "YEAR_A").Entry()
	if err != nil {
		t.Fatal(err)
	}
	other, err := weekdayDay("2019-11-05", "YEAR_C").Entry()
	if err != nil || first.Key != other.Key {
		t.Fatalf("weekday Gospel incorrectly depends on cycle: %+v %+v %v", first, other, err)
	}
	sunday := CalendarDay{Date: "2026-11-08", Observance: "ordinary_time_32_sunday", Temporal: "ordinary_time_32_sunday", Rank: "SUNDAY", SundayCycle: "YEAR_A", WeekdayCycle: "YEAR_2", GospelEntry: "temporal:ordinary_time_32_sunday:YEAR_A"}
	first, err = sunday.Entry()
	if err != nil {
		t.Fatal(err)
	}
	sunday.Date, sunday.SundayCycle, sunday.GospelEntry = "2022-11-06", "YEAR_C", "temporal:ordinary_time_32_sunday:YEAR_C"
	other, err = sunday.Entry()
	if err != nil || first.Key == other.Key {
		t.Fatalf("Sunday cycles conflated: %+v %+v %v", first, other, err)
	}
	memorial := CalendarDay{Date: "2026-11-11", Observance: "martin_of_tours_bishop", Temporal: "ordinary_time_32_wednesday", Rank: "MEMORIAL", SundayCycle: "YEAR_A", WeekdayCycle: "YEAR_2", GospelEntry: "observance:martin_of_tours_bishop"}
	own, err := memorial.Entry()
	if err != nil || own.Displaced != "ordinary_time_32_wednesday" {
		t.Fatalf("proper memorial: %+v %v", own, err)
	}
	memorial.GospelEntry = "temporal:ordinary_time_32_wednesday"
	shared, err := memorial.Entry()
	if err != nil || own.Key == shared.Key {
		t.Fatalf("memorial conflated with temporal day: %+v %+v %v", own, shared, err)
	}
}

func TestWeekdayMemorialSharesWeekdayEntryAcrossYears(t *testing.T) {
	weekday := CalendarDay{Date: "2025-01-22", Observance: "ordinary_time_2_wednesday", Temporal: "ordinary_time_2_wednesday", Rank: "WEEKDAY", SundayCycle: "YEAR_C", WeekdayCycle: "YEAR_1", GospelEntry: "temporal:ordinary_time_2_wednesday"}
	memorial := CalendarDay{Date: "2026-01-21", Observance: "agnes_of_rome_virgin", Temporal: "ordinary_time_2_wednesday", Rank: "MEMORIAL", SundayCycle: "YEAR_A", WeekdayCycle: "YEAR_2", GospelEntry: "temporal:ordinary_time_2_wednesday"}
	a, err := weekday.Entry()
	if err != nil {
		t.Fatal(err)
	}
	b, err := memorial.Entry()
	if err != nil || a.Key != b.Key {
		t.Fatalf("weekday entry not shared: %+v %+v %v", a, b, err)
	}
}

func TestGospelEntryMustMatchTheDay(t *testing.T) {
	for _, bad := range []string{"", "temporal:ordinary_time_30_tuesday", "observance:ordinary_time_31_tuesday:YEAR_B", "temporal:ordinary_time_31_tuesday:day", "proper:all_saints:day"} {
		d := weekdayDay("2026-11-03", "YEAR_A")
		d.GospelEntry = bad
		if _, err := d.Entry(); err == nil {
			t.Fatalf("Gospel entry %q accepted", bad)
		}
	}
}

func TestAssignedMassIsRejected(t *testing.T) {
	d := weekdayDay("2026-11-03", "YEAR_A")
	d.Mass = &MassAssignment{Formulary: "ash_wednesday", Celebration: "Lễ Tro"}
	if _, err := d.Entry(); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatal("assigned Mass accepted", err)
	}
}
