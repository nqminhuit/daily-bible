package lectionary

import (
	"sort"
	"strings"
	"time"
)

func GenerateYear(year int) map[string]DayInfo {
	result := map[string]DayInfo{}

	easter := Easter(year)
	ash := easter.AddDate(0, 0, -46)
	pentecost := easter.AddDate(0, 0, 49)
	advent := startOfAdvent(year)

	start := time.Date(year, 1, 1, 0, 0, 0, 0, utcLoc)
	end := time.Date(year, 12, 31, 0, 0, 0, 0, utcLoc)

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		season := seasonOf(d, ash, easter, pentecost, advent)
		key := d.Format("2006-01-02")
		weekday := strings.ToLower(d.Weekday().String()[:3])

		dayInfo := DayInfo{
			Season:      season,
			SundayCycle: sundayCycle(d),
			Weekday:     weekday,
			MonthDay:    d.Format("01-02"),
		}
		if season == SeasonOrdinary {
			dayInfo.WeekdayCycle = weekdayCycle(year)
		}
		result[key] = dayInfo
	}

	return result
}

func applyWeekNumbers(calendar map[string]DayInfo) {
	dates := make([]string, 0, len(calendar))
	for d := range calendar {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	var (
		adventWeek    int
		christmasWeek int
		lentWeek      int
		easterWeek    int
		ordWeek       int
		ordStarted    bool
		prevSeason    Season = ""
		pastLent      bool
		adventStart   int = -1
	)

	for i, key := range dates {
		day := calendar[key]
		d, _ := time.Parse("2006-01-02", key)
		isSunday := d.Weekday() == time.Sunday

		switch day.Season {
		case SeasonAdvent:
			if prevSeason != SeasonAdvent {
				adventWeek = 1
			} else if isSunday {
				adventWeek++
			}
			day.WeekOfSeason = adventWeek
			if adventStart == -1 {
				adventStart = i
			}

		case SeasonChristmas:
			if prevSeason == SeasonAdvent {
				christmasWeek = 0
			} else if prevSeason != SeasonChristmas {
				christmasWeek = 1
			} else if isSunday {
				christmasWeek++
			}
			day.WeekOfSeason = christmasWeek

		case SeasonLent:
			if prevSeason != SeasonLent {
				lentWeek = 0
			} else if isSunday {
				lentWeek++
			}
			day.WeekOfSeason = lentWeek
			pastLent = true

		case SeasonEaster:
			if prevSeason != SeasonEaster {
				easterWeek = 1
			} else if isSunday {
				easterWeek++
			}
			day.WeekOfSeason = easterWeek

		case SeasonOrdinary:
			if !pastLent {
				if !ordStarted {
					ordStarted = true
					ordWeek = 1
				} else if isSunday {
					ordWeek++
				}
				day.WeekOfSeason = ordWeek
			}
		}

		calendar[key] = day
		prevSeason = day.Season
	}

	if adventStart == -1 {
		return
	}

	ordWeek = 34
	for i := adventStart - 1; i >= 0; i-- {
		key := dates[i]
		day := calendar[key]

		if day.Season != SeasonOrdinary {
			break
		}

		day.WeekOfSeason = ordWeek
		calendar[key] = day

		if day.Weekday == "sun" {
			ordWeek--
			if ordWeek == 0 {
				break
			}
		}
	}
}

func lectionaryPass(calendar map[string]DayInfo) {
	for d, day := range calendar {
		day.LectionaryKey = day.lectionaryKey()
		calendar[d] = day
	}
}

func GenerateLectionary(year int) map[string]string {
	cal := GenerateYear(year)
	applyWeekNumbers(cal)
	lectionaryPass(cal)
	result := make(map[string]string, len(cal))
	for date, info := range cal {
		result[date] = info.LectionaryKey
	}
	return result
}
