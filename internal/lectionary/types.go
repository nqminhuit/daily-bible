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
	Season       Season `json:"season"`
	SundayCycle  string `json:"sunday_cycle"`
	WeekdayCycle string `json:"weekday_cycle"`
	Weekday      string `json:"weekday"`
	WeekOfSeason int    `json:"week_of_season"`
	LectionaryKey string `json:"lectionary_key"`
}

func (d DayInfo) lectionaryKey() string {
	if d.Season == SeasonEaster && d.WeekOfSeason == 1 && d.Weekday == "sun" {
		return fmt.Sprintf("easter_sunday_%s", d.SundayCycle)
	}
	if d.Weekday == "sun" {
		return fmt.Sprintf("%s_%d_sun_%s", d.Season, d.WeekOfSeason, d.SundayCycle)
	}
	if d.WeekdayCycle == "" {
		return fmt.Sprintf("%s_%d_%s", d.Season, d.WeekOfSeason, d.Weekday)
	}
	return fmt.Sprintf("%s_%d_%s_%s", d.Season, d.WeekOfSeason, d.Weekday, d.WeekdayCycle)
}
