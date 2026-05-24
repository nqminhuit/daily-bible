package lectionary

import (
	"testing"
	"time"
)

func TestEaster(t *testing.T) {
	tests := []struct {
		year     int
		expected time.Time
	}{
		{2000, time.Date(2000, 4, 23, 0, 0, 0, 0, utcLoc)},
		{2001, time.Date(2001, 4, 15, 0, 0, 0, 0, utcLoc)},
		{2002, time.Date(2002, 3, 31, 0, 0, 0, 0, utcLoc)},
		{2003, time.Date(2003, 4, 20, 0, 0, 0, 0, utcLoc)},
		{2004, time.Date(2004, 4, 11, 0, 0, 0, 0, utcLoc)},
		{2005, time.Date(2005, 3, 27, 0, 0, 0, 0, utcLoc)},
		{2006, time.Date(2006, 4, 16, 0, 0, 0, 0, utcLoc)},
		{2007, time.Date(2007, 4, 8, 0, 0, 0, 0, utcLoc)},
		{2008, time.Date(2008, 3, 23, 0, 0, 0, 0, utcLoc)},
		{2009, time.Date(2009, 4, 12, 0, 0, 0, 0, utcLoc)},
		{2010, time.Date(2010, 4, 4, 0, 0, 0, 0, utcLoc)},
		{2011, time.Date(2011, 4, 24, 0, 0, 0, 0, utcLoc)},
		{2012, time.Date(2012, 4, 8, 0, 0, 0, 0, utcLoc)},
		{2013, time.Date(2013, 3, 31, 0, 0, 0, 0, utcLoc)},
		{2014, time.Date(2014, 4, 20, 0, 0, 0, 0, utcLoc)},
		{2015, time.Date(2015, 4, 5, 0, 0, 0, 0, utcLoc)},
		{2016, time.Date(2016, 3, 27, 0, 0, 0, 0, utcLoc)},
		{2017, time.Date(2017, 4, 16, 0, 0, 0, 0, utcLoc)},
		{2018, time.Date(2018, 4, 1, 0, 0, 0, 0, utcLoc)},
		{2019, time.Date(2019, 4, 21, 0, 0, 0, 0, utcLoc)},
		{2020, time.Date(2020, 4, 12, 0, 0, 0, 0, utcLoc)},
		{2021, time.Date(2021, 4, 4, 0, 0, 0, 0, utcLoc)},
		{2022, time.Date(2022, 4, 17, 0, 0, 0, 0, utcLoc)},
		{2023, time.Date(2023, 4, 9, 0, 0, 0, 0, utcLoc)},
		{2024, time.Date(2024, 3, 31, 0, 0, 0, 0, utcLoc)},
		{2025, time.Date(2025, 4, 20, 0, 0, 0, 0, utcLoc)},
	}

	for _, test := range tests {
		result := Easter(test.year)
		if !result.Equal(test.expected) {
			t.Errorf("Easter(%d) = %v, want %v", test.year, result, test.expected)
		}
	}
}

func TestStartOfAdvent(t *testing.T) {
	tests := []struct {
		year     int
		expected time.Time
	}{
		{2023, time.Date(2023, 12, 3, 0, 0, 0, 0, utcLoc)},
		{2024, time.Date(2024, 12, 1, 0, 0, 0, 0, utcLoc)},
		{2025, time.Date(2025, 11, 30, 0, 0, 0, 0, utcLoc)},
	}

	for _, test := range tests {
		result := startOfAdvent(test.year)
		if !result.Equal(test.expected) {
			t.Errorf("startOfAdvent(%d) = %v, want %v", test.year, result, test.expected)
		}
	}
}

func TestEpiphany(t *testing.T) {
	tests := []struct {
		year     int
		expected time.Time
	}{
		{2023, time.Date(2023, 1, 8, 0, 0, 0, 0, utcLoc)},
		{2024, time.Date(2024, 1, 7, 0, 0, 0, 0, utcLoc)},
		{2025, time.Date(2025, 1, 5, 0, 0, 0, 0, utcLoc)},
	}

	for _, test := range tests {
		result := epiphany(test.year)
		if !result.Equal(test.expected) {
			t.Errorf("epiphany(%d) = %v, want %v", test.year, result, test.expected)
		}
	}
}

func TestBaptismOfLord(t *testing.T) {
	tests := []struct {
		year     int
		expected time.Time
	}{
		{2023, time.Date(2023, 1, 9, 0, 0, 0, 0, utcLoc)},
		{2024, time.Date(2024, 1, 8, 0, 0, 0, 0, utcLoc)},
		{2025, time.Date(2025, 1, 12, 0, 0, 0, 0, utcLoc)},
	}

	for _, test := range tests {
		result := baptismOfLord(test.year)
		if !result.Equal(test.expected) {
			t.Errorf("baptismOfLord(%d) = %v, want %v", test.year, result, test.expected)
		}
	}
}

func TestLiturgicalYear(t *testing.T) {
	tests := []struct {
		date     time.Time
		expected int
	}{
		{time.Date(2023, 12, 4, 0, 0, 0, 0, utcLoc), 2024},
		{time.Date(2023, 12, 2, 0, 0, 0, 0, utcLoc), 2023},
		{time.Date(2024, 1, 1, 0, 0, 0, 0, utcLoc), 2024},
	}

	for _, test := range tests {
		result := liturgicalYear(test.date)
		if result != test.expected {
			t.Errorf("liturgicalYear(%v) = %d, want %d", test.date, result, test.expected)
		}
	}
}

func TestSundayCycle(t *testing.T) {
	tests := []struct {
		date     time.Time
		expected string
	}{
		{time.Date(2023, 12, 4, 0, 0, 0, 0, utcLoc), "B"},
		{time.Date(2024, 1, 1, 0, 0, 0, 0, utcLoc), "B"},
		{time.Date(2025, 1, 1, 0, 0, 0, 0, utcLoc), "C"},
	}

	for _, test := range tests {
		result := sundayCycle(test.date)
		if result != test.expected {
			t.Errorf("sundayCycle(%v) = %s, want %s", test.date, result, test.expected)
		}
	}
}

func TestLectionaryKey(t *testing.T) {
	tests := []struct {
		day      DayInfo
		expected string
	}{
		{DayInfo{Season: SeasonEaster, WeekOfSeason: 1, Weekday: "sun", SundayCycle: "A"}, "easter_sunday_A"},
		{DayInfo{Season: SeasonEaster, WeekOfSeason: 1, Weekday: "sun", SundayCycle: "B"}, "easter_sunday_B"},
		{DayInfo{Season: SeasonEaster, WeekOfSeason: 1, Weekday: "sun", SundayCycle: "C"}, "easter_sunday_C"},
		{DayInfo{Season: SeasonEaster, WeekOfSeason: 2, Weekday: "sun", SundayCycle: "A"}, "easter_2_sun_A"},
		{DayInfo{Season: SeasonOrdinary, WeekOfSeason: 1, Weekday: "mon", WeekdayCycle: "I"}, "ordinary_1_mon_I"},
		{DayInfo{Season: SeasonLent, WeekOfSeason: 1, Weekday: "mon", WeekdayCycle: ""}, "lent_1_mon"},
		{DayInfo{Season: SeasonLent, WeekOfSeason: 4, Weekday: "thu", MonthDay: "03-19"}, "lent_4_thu_03-19"},
		{DayInfo{Season: SeasonChristmas, WeekOfSeason: 1, Weekday: "mon", MonthDay: "01-01"}, "christmas_1_mon_01-01"},
		{DayInfo{Season: SeasonOrdinary, WeekOfSeason: 1, Weekday: "sun", SundayCycle: "A"}, "ordinary_1_sun_A"},
	}

	for _, tt := range tests {
		got := tt.day.lectionaryKey()
		if got != tt.expected {
			t.Errorf("lectionaryKey(%+v) = %q, want %q", tt.day, got, tt.expected)
		}
	}
}

func TestWeekdayCycle(t *testing.T) {
	tests := []struct {
		year     int
		expected string
	}{
		{2023, "I"},
		{2024, "II"},
		{2025, "I"},
	}

	for _, test := range tests {
		result := weekdayCycle(test.year)
		if result != test.expected {
			t.Errorf("weekdayCycle(%d) = %s, want %s", test.year, result, test.expected)
		}
	}
}

func TestSeasonOf(t *testing.T) {
	year := 2024
	easter := Easter(year)
	ash := easter.AddDate(0, 0, -46)
	pentecost := easter.AddDate(0, 0, 49)
	advent := startOfAdvent(year)

	tests := []struct {
		date     time.Time
		expected Season
	}{
		{time.Date(2024, 11, 25, 0, 0, 0, 0, utcLoc), SeasonOrdinary},
		{time.Date(2023, 12, 26, 0, 0, 0, 0, utcLoc), SeasonChristmas},
		{time.Date(2024, 2, 15, 0, 0, 0, 0, utcLoc), SeasonLent},
		{easter, SeasonEaster},
		{pentecost.AddDate(0, 0, -1), SeasonEaster},
		{pentecost.AddDate(0, 0, 1), SeasonOrdinary},
	}

	for _, test := range tests {
		result := seasonOf(test.date, ash, easter, pentecost, advent)
		if result != test.expected {
			t.Errorf("seasonOf(%v) = %s, want %s", test.date, result, test.expected)
		}
	}
}
