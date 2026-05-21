package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestImportCalendar(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS calendar (
			date           TEXT NOT NULL PRIMARY KEY,
			lectionary_key TEXT NOT NULL,
			season         TEXT NOT NULL,
			sunday_cycle   TEXT NOT NULL,
			weekday        TEXT NOT NULL,
			weekday_cycle  TEXT NOT NULL,
			week_of_season INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	cal := map[string]CalendarEntry{
		"2026-03-22": {
			LectionaryKey: "lent_5_sun_A",
			Season:        "lent",
			SundayCycle:   "A",
			WeekOfSeason:  5,
			Weekday:       "sun",
			WeekdayCycle:  "",
		},
		"2026-01-26": {
			LectionaryKey: "ordinary_3_mon_II",
			Season:        "ordinary",
			SundayCycle:   "",
			WeekOfSeason:  3,
			Weekday:       "mon",
			WeekdayCycle:  "II",
		},
	}

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "calendar.json")
	data, err := json.Marshal(cal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	importCalendar(t, db, jsonPath)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM calendar").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}

	var key string
	err = db.QueryRow("SELECT lectionary_key FROM calendar WHERE date = ?", "2026-03-22").Scan(&key)
	if err != nil {
		t.Fatal(err)
	}
	if key != "lent_5_sun_A" {
		t.Fatalf("expected lent_5_sun_A, got %q", key)
	}
}

func importCalendar(t *testing.T, db *sql.DB, jsonPath string) {
	t.Helper()
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var cal map[string]CalendarEntry
	if err := json.Unmarshal(data, &cal); err != nil {
		t.Fatalf("parse json: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO calendar
			(date, lectionary_key, season, sunday_cycle, weekday, weekday_cycle, week_of_season)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	defer stmt.Close()

	for date, entry := range cal {
		if _, err := stmt.Exec(
			date,
			entry.LectionaryKey,
			entry.Season,
			entry.SundayCycle,
			entry.Weekday,
			entry.WeekdayCycle,
			entry.WeekOfSeason,
		); err != nil {
			t.Logf("insert %s: %v", date, err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestImportCalendarIgnoresDuplicates(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS calendar (
			date           TEXT NOT NULL PRIMARY KEY,
			lectionary_key TEXT NOT NULL,
			season         TEXT NOT NULL,
			sunday_cycle   TEXT NOT NULL,
			weekday        TEXT NOT NULL,
			weekday_cycle  TEXT NOT NULL,
			week_of_season INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	cal := map[string]CalendarEntry{
		"2026-03-22": {
			LectionaryKey: "lent_5_sun_A",
			Season:        "lent",
			SundayCycle:   "A",
			WeekOfSeason:  5,
			Weekday:       "sun",
			WeekdayCycle:  "",
		},
	}

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "calendar.json")
	data, err := json.Marshal(cal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	importCalendar(t, db, jsonPath)
	importCalendar(t, db, jsonPath)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM calendar").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after re-import, got %d", count)
	}
}
