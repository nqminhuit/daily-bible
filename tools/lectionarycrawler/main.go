package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/constants"
	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/lectionary"
	"github.com/minh/daily-bible/internal/parser"
)

func main() {
	var years string
	flag.StringVar(&years, "years", "", "comma-separated list of years (e.g. 2026,2027)")
	flag.Parse()

	if years == "" {
		log.Fatal("--years is required (e.g. -years=2026,2027)")
	}

	db, err := dbpkg.Open(constants.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	yearStrs := strings.Split(years, ",")
	for _, ys := range yearStrs {
		ys = strings.TrimSpace(ys)
		year, err := strconv.Atoi(ys)
		if err != nil {
			log.Fatalf("invalid year %q: %v", ys, err)
		}
		if err := importYear(db, year); err != nil {
			log.Fatalf("import year %d: %v", year, err)
		}
	}

	jobs, err := missingGospelRefs(db)
	if err != nil {
		log.Fatalf("query missing keys: %v", err)
	}
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

type dateKey struct {
	key  string
	date string
}

func importYear(db *sql.DB, year int) error {
	cal := lectionary.GenerateLectionary(year)
	log.Printf("generated %d dates for year %d", len(cal), year)

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
	for date, key := range cal {
		result, err := stmt.Exec(date, key)
		if err != nil {
			log.Printf("insert %s: %v", date, err)
			continue
		}
		if n, _ := result.RowsAffected(); n > 0 {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("commit: %w", err)
	}
	log.Printf("imported %d dates for year %d", inserted, year)
	return nil
}

func missingGospelRefs(db *sql.DB) ([]dateKey, error) {
	rows, err := db.Query(`
		SELECT DISTINCT dr.lectionary_key, MIN(dr.date)
		FROM daily_readings dr
		WHERE dr.gospel_ref IS NULL
		GROUP BY dr.lectionary_key
		ORDER BY MIN(dr.date)
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []dateKey
	for rows.Next() {
		var j dateKey
		if err := rows.Scan(&j.key, &j.date); err != nil {
			log.Printf("scan: %v", err)
			continue
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
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
			if resp.StatusCode == http.StatusNotFound {
				return "", lastErr
			}
			continue
		}

		_, ref, err := parser.ExtractGospel(string(body))
		return ref, err
	}
	return "", lastErr
}
