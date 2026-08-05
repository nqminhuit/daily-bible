package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/constants"
	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/lectionary"
	"github.com/minh/daily-bible/internal/parser"
)

func main() {
	var years, dates string
	flag.StringVar(&years, "years", "", "comma-separated list of years to import and crawl (e.g. 2026,2027)")
	flag.StringVar(&dates, "dates", "", "comma-separated list of specific dates to crawl (YYYY-MM-DD), importing their years if needed")
	flag.Parse()

	if years == "" && dates == "" {
		log.Fatal("--years or --dates is required (e.g. -years=2026,2027 or -dates=2026-08-04)")
	}

	db, err := dbpkg.Open(constants.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := importRequestedYears(db, years, dates); err != nil {
		log.Fatal(err)
	}

	jobs, err := requestedGospelRefs(db, dates)
	if err != nil {
		log.Fatalf("query missing readings: %v", err)
	}
	log.Printf("found %d readings to populate", len(jobs))

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

func importRequestedYears(db *sql.DB, years, dates string) error {
	yearSet := map[int]struct{}{}

	for _, ys := range strings.Split(years, ",") {
		ys = strings.TrimSpace(ys)
		if ys == "" {
			continue
		}
		year, err := strconv.Atoi(ys)
		if err != nil {
			return fmt.Errorf("invalid year %q: %w", ys, err)
		}
		yearSet[year] = struct{}{}
	}

	parsedDates, err := parseDates(dates)
	if err != nil {
		return err
	}
	for _, d := range parsedDates {
		yearSet[d.Year()] = struct{}{}
	}

	yearsToImport := make([]int, 0, len(yearSet))
	for year := range yearSet {
		yearsToImport = append(yearsToImport, year)
	}
	sort.Ints(yearsToImport)

	for _, year := range yearsToImport {
		if err := importYear(db, year); err != nil {
			return fmt.Errorf("import year %d: %w", year, err)
		}
	}
	return nil
}

func parseDates(dates string) ([]time.Time, error) {
	if strings.TrimSpace(dates) == "" {
		return nil, nil
	}

	parts := strings.Split(dates, ",")
	result := make([]time.Time, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		date := strings.TrimSpace(part)
		if date == "" {
			continue
		}
		parsed, err := time.Parse("2006-01-02", date)
		if err != nil {
			return nil, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", date)
		}
		key := parsed.Format("2006-01-02")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, parsed)
	}
	return result, nil
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

func requestedGospelRefs(db *sql.DB, dates string) ([]dateKey, error) {
	parsedDates, err := parseDates(dates)
	if err != nil {
		return nil, err
	}
	if len(parsedDates) == 0 {
		return missingGospelRefs(db)
	}
	return missingGospelRefsForDates(db, parsedDates)
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

func missingGospelRefsForDates(db *sql.DB, dates []time.Time) ([]dateKey, error) {
	jobs := make([]dateKey, 0, len(dates))
	for _, d := range dates {
		date := d.Format("2006-01-02")
		var j dateKey
		err := db.QueryRow(`
			SELECT lectionary_key, date
			FROM daily_readings
			WHERE date = ? AND gospel_ref IS NULL
		`, date).Scan(&j.key, &j.date)
		if err == sql.ErrNoRows {
			log.Printf("skip date=%s: reading is already populated or does not exist", date)
			continue
		}
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
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
