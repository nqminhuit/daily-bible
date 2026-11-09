package main

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/minh/daily-bible/internal/reading"
)

func TestConflictingSourceRequiresExplicitCalendarReview(t *testing.T) {
	db := newCrawlerTestDB(t)
	date := "2026-11-15"
	if _, err := db.Exec(readingsSchema + `; INSERT INTO daily_readings VALUES ('2026-11-15','ordinary_33_sun_A','Lc 9,23-26')`); err != nil {
		t.Fatal(err)
	}
	// Simulate an earlier accepted source. A changed observation must also
	// invalidate previously matching proof, not silently keep serving it.
	if err := storeVerifiedReading(t.Context(), db, date, "Lc 9,23-26", reading.SourceURL(date), false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE daily_reading_sources SET celebration='Previously reviewed local celebration' WHERE date=?`, date); err != nil {
		t.Fatal(err)
	}
	err := storeVerifiedReading(t.Context(), db, date, "Mt 25,14-30", reading.SourceURL(date), false)
	if !errors.Is(err, errReferenceConflict) {
		t.Fatalf("expected review conflict, got %v", err)
	}
	var stored, observed string
	var label sql.NullString
	if err := db.QueryRow(`SELECT dr.gospel_ref,s.gospel_ref,s.celebration FROM daily_readings dr JOIN daily_reading_sources s USING(date) WHERE date=?`, date).Scan(&stored, &observed, &label); err != nil {
		t.Fatal(err)
	}
	if stored != "Lc 9,23-26" || observed != "Mt 25,14-30" || label.Valid {
		t.Fatalf("incorrect conflict evidence: %q %q %+v", stored, observed, label)
	}
	if _, err := reading.Lookup(t.Context(), db, date); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("conflict remained available: %v", err)
	}
	jobs, err := pendingDates(t.Context(), db, []string{"2026"}, crawlWindow{from: date, to: date})
	if err != nil || len(jobs) != 1 {
		t.Fatalf("unresolved conflict skipped: %+v %v", jobs, err)
	}
	// Only an explicit reviewed operation can accept the source replacement.
	if err := storeVerifiedReading(t.Context(), db, date, "Mt 25,14-30", reading.SourceURL(date), true); err != nil {
		t.Fatal(err)
	}
	got, err := reading.Lookup(t.Context(), db, date)
	if err != nil || got.Ref != "Mt 25,14-30" || got.Celebration != "" {
		t.Fatalf("reviewed resolution: %+v %v", got, err)
	}
}

func TestCrawlContinuesAfterUnresolvedConflict(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema + `; INSERT INTO daily_readings VALUES ('2026-04-05','same_key','Lc 1,1'),('2026-04-06','same_key',NULL)`); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		page := gospelPage
		if strings.HasSuffix(r.URL.Path, "06.html") {
			page = strings.ReplaceAll(page, "ngày 05", "ngày 06")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(page)), Header: make(http.Header)}, nil
	})}
	err := populateRefs(t.Context(), db, client, []dateKey{{date: "2026-04-05", key: "same_key"}, {date: "2026-04-06", key: "same_key"}}, false)
	if !errors.Is(err, errReferenceConflict) {
		t.Fatalf("conflict was not reported: %v", err)
	}
	if _, err := reading.Lookup(t.Context(), db, "2026-04-05"); err != sql.ErrNoRows {
		t.Fatalf("unreviewed conflict served: %v", err)
	}
	if _, err := reading.Lookup(t.Context(), db, "2026-04-06"); err != nil {
		t.Fatalf("later date was not verified: %v", err)
	}
}
