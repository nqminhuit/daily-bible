package lectionary

import (
	"fmt"
	"time"
)

type Season string

const (
	SeasonAdvent    Season = "advent"
	SeasonChristmas Season = "christmas"
	SeasonLent      Season = "lent"
	SeasonEaster    Season = "easter"
	SeasonOrdinary  Season = "ordinary"
)

var utcLoc = time.UTC

type DayInfo struct {
	Season        Season `json:"season"`
	SundayCycle   string `json:"sunday_cycle"`
	WeekdayCycle  string `json:"weekday_cycle"`
	Weekday       string `json:"weekday"`
	WeekOfSeason  int    `json:"week_of_season"`
	MonthDay      string `json:"month_day"`
	LectionaryKey string `json:"lectionary_key"`
}

var fixedCelebrationDates = map[string]struct{}{
	"01-01": {}, // Mary, Mother of God
	"01-06": {}, // Epiphany (where not transferred)
	"02-02": {}, // Presentation of the Lord
	"02-11": {}, // Our Lady of Lourdes
	"03-19": {}, // Saint Joseph
	"03-25": {}, // Annunciation
	"06-24": {}, // Nativity of John the Baptist
	"06-29": {}, // Saints Peter and Paul
	"08-06": {}, // Transfiguration
	"08-15": {}, // Assumption
	"09-14": {}, // Exaltation of the Holy Cross
	"11-01": {}, // All Saints
	"11-02": {}, // Commemoration of All the Faithful Departed
	"12-08": {}, // Immaculate Conception
	"12-12": {}, // Our Lady of Guadalupe
	"12-25": {}, // Nativity of the Lord
	"12-26": {}, // Saint Stephen
	"12-27": {}, // Saint John, Apostle and Evangelist
	"12-28": {}, // Holy Innocents
}

func (d DayInfo) hasDateSpecificReadings() bool {
	if d.MonthDay == "" || d.Weekday == "sun" {
		return false
	}
	if d.Season == SeasonChristmas {
		return true
	}
	_, ok := fixedCelebrationDates[d.MonthDay]
	return ok
}

func (d DayInfo) lectionaryKey() string {
	if d.Season == SeasonEaster && d.WeekOfSeason == 1 && d.Weekday == "sun" {
		return fmt.Sprintf("easter_sunday_%s", d.SundayCycle)
	}
	if d.hasDateSpecificReadings() {
		if d.WeekdayCycle == "" {
			return fmt.Sprintf("%s_%d_%s_%s", d.Season, d.WeekOfSeason, d.Weekday, d.MonthDay)
		}
		return fmt.Sprintf("%s_%d_%s_%s_%s", d.Season, d.WeekOfSeason, d.Weekday, d.MonthDay, d.WeekdayCycle)
	}
	if d.Weekday == "sun" {
		return fmt.Sprintf("%s_%d_sun_%s", d.Season, d.WeekOfSeason, d.SundayCycle)
	}
	if d.WeekdayCycle == "" {
		return fmt.Sprintf("%s_%d_%s", d.Season, d.WeekOfSeason, d.Weekday)
	}
	return fmt.Sprintf("%s_%d_%s_%s", d.Season, d.WeekOfSeason, d.Weekday, d.WeekdayCycle)
}
