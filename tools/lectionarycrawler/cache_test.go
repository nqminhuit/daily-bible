package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/lectionary"
	"github.com/minh/daily-bible/internal/reading"
)

func cacheFixture(t *testing.T, reviewed bool) (*http.Client, func(bool, bool) (cacheStats, error)) {
	t.Helper()
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema + `; CREATE TABLE verses(book TEXT,chapter INTEGER,verse INTEGER,verse_suffix TEXT,text TEXT)`); err != nil {
		t.Fatal(err)
	}
	manifest := reading.CalendarManifest{GospelEntrySchema: lectionary.GospelEntrySchemaVersion, CalendarID: "roman-general", RulesVersion: "fixture-v1", SourceURL: "https://example.test/calendar", Days: []lectionary.CalendarDay{
		{Date: "2025-11-09", Observance: "dedication_of_the_lateran_basilica", Rank: "FEAST", Temporal: "ordinary_time_32_sunday", SundayCycle: "YEAR_C", WeekdayCycle: "YEAR_1", GospelEntry: "observance:dedication_of_the_lateran_basilica"},
		{Date: "2027-11-09", Observance: "dedication_of_the_lateran_basilica", Rank: "FEAST", Temporal: "ordinary_time_32_tuesday", SundayCycle: "YEAR_B", WeekdayCycle: "YEAR_1", GospelEntry: "observance:dedication_of_the_lateran_basilica"},
	}}
	if err := reading.ImportCalendar(t.Context(), db, manifest); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		page := `<html><head><title>Lời Chúa ngày 9 tháng 11 2025</title></head><body><section><div class="section__content"><p>✠Bài trích Tin Mừng Chúa Giê-su Ki-tô theo thánh Gio-an. Ga 2,13-22</p><p><sup>13</sup>Text</p></div></section></body></html>`
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(page))}, nil
	})}
	run := func(offline, refresh bool) (cacheStats, error) {
		return populateCacheFirst(t.Context(), db, client, []dateKey{{date: "2025-11-09"}, {date: "2027-11-09"}}, crawlWindow{calendar: manifest.CalendarID, offline: offline, refresh: refresh})
	}
	t.Cleanup(func() { client.CloseIdleConnections() })
	if reviewed {
		if err := reading.ReviewEntry(t.Context(), db, reading.EntryReview{CalendarID: manifest.CalendarID, Date: "2025-11-09", Ref: "Ga 2,13-22", EvidenceURL: "https://example.test/lectionary", Note: "Reviewed invariant proper Gospel for daytime Mass"}); err != nil {
			t.Fatal(err)
		}
		for verse := 13; verse <= 22; verse++ {
			if _, err := db.Exec(`INSERT INTO verses VALUES ('Ga',2,?,'','text')`, verse); err != nil {
				t.Fatal(err)
			}
		}
	}
	return client, run
}

func TestWarmCacheUsesZeroHTTPAcrossYears(t *testing.T) {
	client, run := cacheFixture(t, true)
	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("warm cache made an HTTP request")
		return nil, nil
	})
	for _, offline := range []bool{false, true} {
		stats, err := run(offline, false)
		if err != nil || stats.CacheHits != 2 || stats.HTTPRequests != 0 || stats.Unresolved != 0 {
			t.Fatalf("stats=%+v err=%v", stats, err)
		}
	}
}
func TestMissingEntryFetchesOnceThenReusesObservationCandidate(t *testing.T) {
	_, run := cacheFixture(t, false)
	stats, err := run(false, false)
	if err == nil || stats.HTTPRequests != 1 || stats.Candidates != 1 || stats.Unresolved != 2 {
		t.Fatalf("first run: %+v %v", stats, err)
	}
	stats, err = run(false, false)
	if err == nil || stats.HTTPRequests != 0 || stats.Candidates != 1 || stats.Unresolved != 2 {
		t.Fatalf("candidate was recrawled/promoted: %+v %v", stats, err)
	}
}
func TestOfflineUnresolvedEntriesNeverCrawl(t *testing.T) {
	client, run := cacheFixture(t, false)
	client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("offline request"); return nil, nil })
	stats, err := run(true, false)
	if err == nil || stats.HTTPRequests != 0 || stats.Unresolved != 2 {
		t.Fatalf("offline: %+v %v", stats, err)
	}
}
func TestSourceObservationHistorySurvivesConflictAndMigration(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema + `; INSERT INTO daily_readings VALUES ('2026-11-09','estimate','Lc 17,1-6')`); err != nil {
		t.Fatal(err)
	}
	if err := storeVerifiedReading(t.Context(), db, "2026-11-09", "Lc 17,1-6", reading.SourceURL("2026-11-09"), false); err != nil {
		t.Fatal(err)
	}
	err := storeVerifiedReading(t.Context(), db, "2026-11-09", "Ga 2,13-22", reading.SourceURL("2026-11-09"), false)
	if !errors.Is(err, errReferenceConflict) {
		t.Fatal(err)
	}
	if err := dbpkg.EnsureReadingCache(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reading_observations`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("history duplicated/overwritten: %d %v", count, err)
	}
}
