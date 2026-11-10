package main

import (
	"net/http"
	"testing"

	"github.com/minh/daily-bible/internal/lectionary"
	"github.com/minh/daily-bible/internal/reading"
)

func TestNationalAssignedMassAndUnsupportedDateDoNotCrawlWrongScope(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema); err != nil {
		t.Fatal(err)
	}
	m := reading.CalendarManifest{GospelEntrySchema: lectionary.GospelEntrySchemaVersion, CalendarID: "vietnam", RulesVersion: "vietnam-national-pilot-fixture", SourceURL: "https://example.test/issuer", Days: []lectionary.CalendarDay{
		{Date: "2026-02-18", Observance: "ash_wednesday", Temporal: "ash_wednesday", Rank: "WEEKDAY", SundayCycle: "YEAR_A", WeekdayCycle: "YEAR_2", Mass: &lectionary.MassAssignment{Formulary: "tet_day_2_ancestors", Celebration: "Mồng Hai Tết"}, SourceURL: "https://example.test/directive", ReviewNote: "National fixture assignment"},
		{Date: "2026-11-15", Observance: "ordinary_time_33_sunday", Temporal: "ordinary_time_33_sunday", Rank: "SUNDAY", SundayCycle: "YEAR_A", WeekdayCycle: "YEAR_2", UnsupportedReason: "National Martyrs formulary unresolved"},
	}}
	if err := reading.ImportCalendar(t.Context(), db, m); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("wrong-scope Vatican crawl"); return nil, nil })}
	stats, err := populateCacheFirst(t.Context(), db, client, []dateKey{{date: "2026-02-18"}, {date: "2026-11-15"}}, crawlWindow{calendar: "vietnam"})
	if err == nil || stats.HTTPRequests != 0 || stats.Unresolved != 2 || stats.Candidates != 0 {
		t.Fatalf("%+v %v", stats, err)
	}
}
