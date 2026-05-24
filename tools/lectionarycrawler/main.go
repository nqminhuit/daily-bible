package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/constants"
	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/parser"
)

type CalendarEntry struct {
	LectionaryKey string `json:"lectionary_key"`
}

func main() {
	var calendarURLs string
	flag.StringVar(&calendarURLs, "calendar-urls", "", "comma-separated list of liturgical calendar JSON URLs")
	flag.Parse()

	if calendarURLs == "" {
		log.Fatal("--calendar-urls is required")
	}

	urls := strings.Split(calendarURLs, ",")
	for i := range urls {
		urls[i] = strings.TrimSpace(urls[i])
	}

	db, err := dbpkg.Open(constants.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	fileCount := 0
	for _, url := range urls {
		if err := importCalendarURL(client, db, url); err != nil {
			log.Fatalf("import %s: %v", url, err)
		}
		fileCount++
	}
	log.Printf("imported %d calendar files", fileCount)

	rows, err := db.Query(`
		SELECT DISTINCT dr.lectionary_key, MIN(dr.date)
		FROM daily_readings dr
		WHERE dr.gospel_ref IS NULL
		GROUP BY dr.lectionary_key
		ORDER BY MIN(dr.date)
	`)
	if err != nil {
		log.Fatalf("query missing keys: %v", err)
	}
	defer rows.Close()

	type job struct {
		key  string
		date string
	}
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.key, &j.date); err != nil {
			log.Printf("scan: %v", err)
			continue
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows iteration: %v", err)
	}
	rows.Close()

	log.Printf("found %d lectionary keys to populate", len(jobs))

	crawlClient := &http.Client{Timeout: 10 * time.Second}

	for i, j := range jobs {
		parts := strings.SplitN(j.date, "-", 3)
		if len(parts) != 3 {
			log.Printf("invalid date %q", j.date)
			continue
		}
		url := fmt.Sprintf(
			"https://www.vaticannews.va/vi/loi-chua-hang-ngay/%s/%s/%s.html",
			parts[0], parts[1], parts[2],
		)

		ref, err := fetchGospelRef(crawlClient, url)
		if err != nil {
			log.Printf("[%d/%d] FAIL key=%s date=%s: %v", i+1, len(jobs), j.key, j.date, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if _, err := db.Exec(
			`UPDATE daily_readings SET gospel_ref = ? WHERE lectionary_key = ? AND gospel_ref IS NULL`,
			ref, j.key,
		); err != nil {
			log.Printf("update %s: %v", j.key, err)
			continue
		}

		log.Printf("[%d/%d] OK key=%s ref=%s", i+1, len(jobs), j.key, ref)
		time.Sleep(300 * time.Millisecond)
	}
}

func importCalendarURL(client *http.Client, db *sql.DB, url string) error {
	data, err := fetchURL(client, url)
	if err != nil {
		return fmt.Errorf("fetch url: %w", err)
	}

	var cal map[string]CalendarEntry
	if err := json.Unmarshal(data, &cal); err != nil {
		return fmt.Errorf("parse json: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO daily_readings (date, lectionary_key) VALUES (?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for date, entry := range cal {
		if _, err := stmt.Exec(date, entry.LectionaryKey); err != nil {
			log.Printf("insert %s: %v", date, err)
			continue
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("commit: %w", err)
	}
	log.Printf("imported %d dates from %s", inserted, url)
	return nil
}

func fetchURL(client *http.Client, rawURL string) ([]byte, error) {
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

func fetchGospelRef(client *http.Client, url string) (string, error) {
	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			time.Sleep(time.Duration(100*(1<<attempt)) * time.Millisecond)
		}
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read body: %w", err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 {
				return "", lastErr
			}
			continue
		}

		_, ref, err := parser.ExtractGospel(string(body))
		return ref, err
	}
	return "", lastErr
}
