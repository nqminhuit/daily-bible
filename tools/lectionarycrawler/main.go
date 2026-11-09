package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/minh/daily-bible/internal/constants"
	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/lectionary"
	"github.com/minh/daily-bible/internal/parser"
)

func databasePath() string {
	if path := os.Getenv("DB_PATH"); path != "" {
		return path
	}
	return constants.DBPath
}

func main() {
	var years string
	flag.StringVar(&years, "years", "", "comma-separated list of years (e.g. 2026,2027)")
	flag.Parse()

	if years == "" {
		log.Fatal("--years is required (e.g. -years=2026,2027)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, years)
	stop()
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, years string) error {
	db, err := dbpkg.Open(databasePath())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	yearStrs := strings.Split(years, ",")
	for _, ys := range yearStrs {
		ys = strings.TrimSpace(ys)
		year, err := strconv.Atoi(ys)
		if err != nil {
			return fmt.Errorf("invalid year %q: %w", ys, err)
		}
		if err := importYear(ctx, db, year); err != nil {
			return fmt.Errorf("import year %d: %w", year, err)
		}
	}

	jobs, err := missingGospelRefs(ctx, db, yearStrs)
	if err != nil {
		return fmt.Errorf("query missing keys: %w", err)
	}
	log.Printf("found %d lectionary keys to populate", len(jobs))

	crawlClient := &http.Client{Timeout: 10 * time.Second}
	defer crawlClient.CloseIdleConnections()
	return populateRefs(ctx, db, crawlClient, jobs, time.Now().Format("2006-01-02"))
}

func populateRefs(ctx context.Context, db *sql.DB, client *http.Client, jobs []dateKey, today string) error {
	for i, j := range jobs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if j.date > today {
			log.Printf("[%d/%d] SKIP key=%s date=%s: date is in the future, not yet published", i+1, len(jobs), j.key, j.date)
			continue
		}

		parts := strings.SplitN(j.date, "-", 3)
		if len(parts) != 3 {
			log.Printf("invalid date %q", j.date)
			continue
		}
		url := fmt.Sprintf(
			"https://www.vaticannews.va/vi/loi-chua-hang-ngay/%s/%s/%s.html",
			parts[0], parts[1], parts[2],
		)

		ref, err := fetchGospelRef(ctx, client, url, j.date)
		if err != nil {
			log.Printf("[%d/%d] FAIL key=%s date=%s: %v", i+1, len(jobs), j.key, j.date, err)
			if err := waitContext(ctx, 500*time.Millisecond); err != nil {
				return err
			}
			continue
		}

		if _, err := db.ExecContext(ctx,
			`UPDATE daily_readings SET gospel_ref = ? WHERE lectionary_key = ? AND gospel_ref IS NULL`,
			ref, j.key,
		); err != nil {
			return fmt.Errorf("update %s: %w", j.key, err)
		}

		log.Printf("[%d/%d] OK key=%s ref=%s", i+1, len(jobs), j.key, ref)
		if err := waitContext(ctx, 300*time.Millisecond); err != nil {
			return err
		}
	}
	return ctx.Err()
}

type dateKey struct {
	key  string
	date string
}

func importYear(ctx context.Context, db *sql.DB, year int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cal := lectionary.GenerateLectionary(year)
	log.Printf("generated %d dates for year %d", len(cal), year)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT OR IGNORE INTO daily_readings (date, lectionary_key) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for date, key := range cal {
		result, err := stmt.ExecContext(ctx, date, key)
		if err != nil {
			return fmt.Errorf("insert %s: %w", date, err)
		}
		n, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("rows affected: %w", err)
		}
		inserted += int(n)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	log.Printf("imported %d dates for year %d", inserted, year)
	return nil
}

func missingGospelRefs(ctx context.Context, db *sql.DB, years []string) ([]dateKey, error) {
	// Build date range filters: date BETWEEN 'YYYY-01-01' AND 'YYYY-12-31'
	var conditions []string
	var args []interface{}
	for _, ys := range years {
		ys = strings.TrimSpace(ys)
		conditions = append(conditions, "dr.date BETWEEN ? AND ?")
		args = append(args, ys+"-01-01", ys+"-12-31")
	}
	dateFilter := "(" + strings.Join(conditions, " OR ") + ")"

	query := fmt.Sprintf(`
		SELECT DISTINCT dr.lectionary_key, MIN(dr.date)
		FROM daily_readings dr
		WHERE dr.gospel_ref IS NULL AND %s
		GROUP BY dr.lectionary_key
		ORDER BY MIN(dr.date)
	`, dateFilter)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []dateKey
	for rows.Next() {
		var j dateKey
		if err := rows.Scan(&j.key, &j.date); err != nil {
			return nil, fmt.Errorf("scan missing gospel ref: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func fetchGospelRef(ctx context.Context, client *http.Client, url string, expectedDate string) (string, error) {
	var lastErr error
	for attempt := range 3 {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if attempt > 0 {
			if err := waitContext(ctx, time.Duration(100*(1<<attempt))*time.Millisecond); err != nil {
				return "", err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		resp, err := client.Do(req)
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

		pageDate, err := parser.ExtractPageDate(string(body))
		if err != nil {
			return "", fmt.Errorf("extract page date: %w", err)
		}
		if pageDate != expectedDate {
			return "", fmt.Errorf("page date mismatch: requested %s, page is for %s", expectedDate, pageDate)
		}

		_, ref, err := parser.ExtractGospel(string(body))
		return ref, err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "", lastErr
}

func waitContext(ctx context.Context, duration time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
