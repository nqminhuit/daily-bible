package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/minh/daily-bible/internal/constants"
	dbpkg "github.com/minh/daily-bible/internal/db"
)

type CalendarEntry struct {
	LectionaryKey string `json:"lectionary_key"`
	Season        string `json:"season"`
	SundayCycle   string `json:"sunday_cycle"`
	WeekOfSeason  int    `json:"week_of_season"`
	Weekday       string `json:"weekday"`
	WeekdayCycle  string `json:"weekday_cycle"`
}

func main() {
	var jsonPath string
	flag.StringVar(&jsonPath, "file", "", "path to liturgical calendar JSON file")
	flag.Parse()

	if jsonPath == "" {
		log.Fatal("--file required")
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		log.Fatalf("read file: %v", err)
	}

	var cal map[string]CalendarEntry
	if err := json.Unmarshal(data, &cal); err != nil {
		log.Fatalf("parse json: %v", err)
	}

	db, err := dbpkg.Open(constants.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO calendar
			(date, lectionary_key, season, sunday_cycle, weekday, weekday_cycle, week_of_season)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		log.Fatalf("prepare: %v", err)
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
			log.Printf("insert %s: %v", date, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}
	log.Printf("imported %d calendar entries", len(cal))
}
