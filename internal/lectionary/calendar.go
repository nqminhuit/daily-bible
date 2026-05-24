package lectionary

import "time"

func Easter(year int) time.Time {
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, utcLoc)
}

func startOfAdvent(year int) time.Time {
	start := time.Date(year, 11, 27, 0, 0, 0, 0, utcLoc)
	offset := (7 - int(start.Weekday())) % 7
	return start.AddDate(0, 0, offset)
}

func epiphany(year int) time.Time {
	d := time.Date(year, 1, 2, 0, 0, 0, 0, utcLoc)
	offset := (7 - int(d.Weekday())) % 7
	return d.AddDate(0, 0, offset)
}

func baptismOfLord(year int) time.Time {
	e := epiphany(year)
	if e.Day() >= 7 {
		return e.AddDate(0, 0, 1)
	}
	return e.AddDate(0, 0, 7)
}

func liturgicalYear(d time.Time) int {
	advent := startOfAdvent(d.Year())
	if !d.Before(advent) {
		return d.Year() + 1
	}
	return d.Year()
}

func sundayCycle(d time.Time) string {
	ly := liturgicalYear(d)
	switch (ly - 2008) % 3 {
	case 0:
		return "A"
	case 1:
		return "B"
	default:
		return "C"
	}
}

func weekdayCycle(year int) string {
	if year%2 == 0 {
		return "II"
	}
	return "I"
}

func seasonOf(d time.Time, ash, easter, pentecost, advent time.Time) Season {
	year := d.Year()
	christmasCurrent := time.Date(year, 12, 25, 0, 0, 0, 0, utcLoc)
	christmasPrev := time.Date(year-1, 12, 25, 0, 0, 0, 0, utcLoc)
	baptism := baptismOfLord(year)

	switch {
	case !d.Before(christmasPrev) && d.Before(baptism.AddDate(0, 0, 1)):
		return SeasonChristmas
	case !d.Before(christmasCurrent):
		return SeasonChristmas
	case !d.Before(ash) && d.Before(easter):
		return SeasonLent
	case !d.Before(easter) && !d.After(pentecost):
		return SeasonEaster
	case !d.Before(advent) && d.Before(christmasCurrent):
		return SeasonAdvent
	default:
		return SeasonOrdinary
	}
}
