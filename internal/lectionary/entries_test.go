package lectionary

import "testing"

func TestGospelIdentityUsesAnnualWeekdaysButThreeYearSundays(t *testing.T) {
	d := CalendarDay{Date: "2026-11-03", Observance: "ordinary_time_31_tuesday", Temporal: "ordinary_time_31_tuesday", Rank: "WEEKDAY", SundayCycle: "YEAR_A", WeekdayCycle: "YEAR_2"}
	first, err := d.Entry()
	if err != nil {
		t.Fatal(err)
	}
	d.Date = "2019-11-05"
	d.SundayCycle = "YEAR_C"
	d.WeekdayCycle = "YEAR_1"
	other, err := d.Entry()
	if err != nil || first.Key != other.Key {
		t.Fatalf("weekday Gospel incorrectly depends on first-reading cycle: %+v %+v %v", first, other, err)
	}
	d.Date = "2026-11-08"
	d.Observance = "ordinary_time_32_sunday"
	d.Temporal = d.Observance
	d.Rank = "SUNDAY"
	d.SundayCycle = "YEAR_A"
	first, err = d.Entry()
	if err != nil {
		t.Fatal(err)
	}
	d.Date = "2022-11-06"
	d.SundayCycle = "YEAR_C"
	other, err = d.Entry()
	if err != nil || first.Key == other.Key {
		t.Fatalf("Sunday cycles conflated: %+v %+v %v", first, other, err)
	}
	d.Date = "2026-11-11"
	d.Observance = "martin_of_tours_bishop"
	d.Temporal = "ordinary_time_32_wednesday"
	d.Rank = "MEMORIAL"
	d.SundayCycle = "YEAR_A"
	memorial, err := d.Entry()
	if err != nil {
		t.Fatal(err)
	}
	d.Observance = d.Temporal
	d.Rank = "WEEKDAY"
	weekday, err := d.Entry()
	if err != nil || memorial.Key == weekday.Key {
		t.Fatalf("memorial conflated with temporal day: %+v %+v %v", memorial, weekday, err)
	}
}
