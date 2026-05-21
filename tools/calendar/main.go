package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

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
	var jsonURL string
	flag.StringVar(&jsonPath, "file", "", "path to liturgical calendar JSON file")
	flag.StringVar(&jsonURL, "url", "", "URL to liturgical calendar JSON file")
	flag.Parse()

	if jsonPath == "" && jsonURL == "" {
		log.Fatal("--file or --url required")
	}

	var data []byte
	var err error

	if jsonURL != "" {
		data, err = fetchURL(jsonURL)
		if err != nil {
			log.Fatalf("fetch url: %v", err)
		}
	} else {
		data, err = os.ReadFile(jsonPath)
		if err != nil {
			log.Fatalf("read file: %v", err)
		}
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

	inserted := 0
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
		} else {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}
	log.Printf("imported %d calendar entries", inserted)
}

func fetchURL(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}
