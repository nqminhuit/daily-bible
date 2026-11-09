package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/reading"
)

func TestExactDateVerificationDoesNotPropagateByKey(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema + `; INSERT INTO daily_readings VALUES
		('2022-11-07','ordinary_32_mon_II','Lc 17,1-6'),
		('2026-11-09','ordinary_32_mon_II','Lc 17,1-6')`); err != nil {
		t.Fatal(err)
	}
	jobs, err := missingGospelRefs(t.Context(), db, []string{"2022", "2026"})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("must check both dates despite identical keys: %+v", jobs)
	}
	if err := storeVerifiedReading(t.Context(), db, "2026-11-09", "Ga 2,13-22", reading.SourceURL("2026-11-09"), true); err != nil {
		t.Fatal(err)
	}
	var previous string
	if err := db.QueryRow(`SELECT gospel_ref FROM daily_readings WHERE date='2022-11-07'`).Scan(&previous); err != nil {
		t.Fatal(err)
	}
	if previous != "Lc 17,1-6" {
		t.Fatalf("reference propagated across years: %s", previous)
	}
	proof, err := reading.Lookup(t.Context(), db, "2026-11-09")
	if err != nil || proof.Ref != "Ga 2,13-22" {
		t.Fatalf("incorrect verification: %+v, %v", proof, err)
	}
	jobs, err = missingGospelRefs(t.Context(), db, []string{"2022", "2026"})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].date != "2022-11-07" {
		t.Fatalf("incorrect pending dates: %+v", jobs)
	}
}

func TestReviewedSourceOverridesCalendarPredictions(t *testing.T) {
	for _, tc := range []struct{ date, key, predicted, observed string }{
		{"2025-11-09", "ordinary_32_sun_C", "Lc 20,27-38", "Ga 2,13-22"}, // feast on an Ordinary Sunday
		{"2026-11-09", "ordinary_32_mon_II", "Lc 17,1-6", "Ga 2,13-22"},
		{"2026-11-01", "ordinary_31_sun_A", "Mt 23,1-12", "Mt 5,1-12a"},
		{"2024-04-08", "easter_2_mon", "Ga 3,1-8", "Lc 1,26-38"}, // transferred Annunciation
	} {
		t.Run(tc.date, func(t *testing.T) {
			db := newCrawlerTestDB(t)
			if _, err := db.Exec(readingsSchema); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO daily_readings VALUES (?,?,?)`, tc.date, tc.key, tc.predicted); err != nil {
				t.Fatal(err)
			}
			if err := storeVerifiedReading(t.Context(), db, tc.date, tc.observed, reading.SourceURL(tc.date), true); err != nil {
				t.Fatal(err)
			}
			got, err := reading.Lookup(t.Context(), db, tc.date)
			if err != nil || got.Ref != tc.observed {
				t.Fatalf("source was not authoritative: %+v, %v", got, err)
			}
		})
	}
}

func TestSourceAndReferenceUpdateAreAtomic(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema + `; INSERT INTO daily_readings VALUES ('2026-11-09','key','Lc 17,1-6');
		CREATE TRIGGER reject_source BEFORE INSERT ON daily_reading_sources BEGIN SELECT RAISE(ABORT, 'proof rejected'); END`); err != nil {
		t.Fatal(err)
	}
	if err := storeVerifiedReading(t.Context(), db, "2026-11-09", "Ga 2,13-22", reading.SourceURL("2026-11-09"), true); err == nil {
		t.Fatal("expected source write failure")
	}
	var ref string
	if err := db.QueryRow(`SELECT gospel_ref FROM daily_readings`).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	if ref != "Lc 17,1-6" {
		t.Fatal("failed transaction partially changed the reading")
	}
}

func TestRefreshWindowAndLegacyReverification(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema + `; INSERT INTO daily_readings VALUES
		('2026-11-08','key','Lc 20,27-38'), ('2026-11-09','key','Lc 17,1-6'),
		('2026-11-10','key','Lc 17,7-10')`); err != nil {
		t.Fatal(err)
	}
	if err := storeVerifiedReading(t.Context(), db, "2026-11-09", "Ga 2,13-22", reading.SourceURL("2026-11-09"), true); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		refresh bool
		want    int
	}{{false, 0}, {true, 1}} {
		jobs, err := pendingDates(t.Context(), db, []string{"2026"}, crawlWindow{from: "2026-11-09", to: "2026-11-09", refresh: tc.refresh})
		if err != nil || len(jobs) != tc.want {
			t.Fatalf("refresh=%v: %+v %v", tc.refresh, jobs, err)
		}
	}
	if _, err := db.Exec(`UPDATE daily_readings SET gospel_ref='Lc 17,1-6' WHERE date='2026-11-09'`); err != nil {
		t.Fatal(err)
	}
	jobs, err := pendingDates(t.Context(), db, []string{"2026"}, crawlWindow{from: "2026-11-08", to: "2026-11-10"})
	if err != nil || len(jobs) != 3 {
		t.Fatalf("stale proof or legacy data skipped: %+v %v", jobs, err)
	}
}

func TestWindowValidation(t *testing.T) {
	for _, w := range []crawlWindow{{from: "2026-11-09"}, {from: "bad", to: "bad"}, {from: "2026-11-10", to: "2026-11-09"}} {
		if _, err := validateWindow("2026", w); err == nil {
			t.Fatalf("accepted invalid window: %+v", w)
		}
	}
	years, err := validateWindow("2020", crawlWindow{from: "2026-12-31", to: "2027-01-02"})
	if err != nil || years != "2026,2027" {
		t.Fatalf("cross-year window: %q %v", years, err)
	}
}

func TestFutureDateRequiresExactPublishedSource(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema + `; INSERT INTO daily_readings VALUES ('2099-04-05','key',NULL)`); err != nil {
		t.Fatal(err)
	}
	jobs, err := pendingDates(t.Context(), db, []string{"2099"}, crawlWindow{from: "2099-04-05", to: "2099-04-05"})
	if err != nil || len(jobs) != 1 {
		t.Fatalf("explicit future window: %+v %v", jobs, err)
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.ReplaceAll(gospelPage, "2026", "2099"))), Header: make(http.Header)}, nil
	})}
	if err := populateRefs(t.Context(), db, client, jobs, false); err != nil {
		t.Fatal(err)
	}
	if _, err := reading.Lookup(t.Context(), db, "2099-04-05"); err != nil {
		t.Fatal(err)
	}
}

func TestImportYearRepairsStaleKeyWithoutBlessingReference(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema + `; INSERT INTO daily_readings VALUES ('2026-11-09','stale_key','Lc 17,1-6')`); err != nil {
		t.Fatal(err)
	}
	if err := importYear(t.Context(), db, 2026); err != nil {
		t.Fatal(err)
	}
	var key, ref string
	if err := db.QueryRow(`SELECT lectionary_key,gospel_ref FROM daily_readings WHERE date='2026-11-09'`).Scan(&key, &ref); err != nil {
		t.Fatal(err)
	}
	if key == "stale_key" || ref != "Lc 17,1-6" {
		t.Fatalf("unexpected regenerated row: %s %s", key, ref)
	}
	if _, err := reading.Lookup(t.Context(), db, "2026-11-09"); err != sql.ErrNoRows {
		t.Fatalf("calendar prediction became verified: %v", err)
	}
}

func TestEnsureSourcesIsIdempotent(t *testing.T) {
	db := newCrawlerTestDB(t)
	for range 2 {
		if err := dbpkg.EnsureReadingSources(context.Background(), db); err != nil {
			t.Fatal(fmt.Errorf("repeat schema: %w", err))
		}
	}
}
