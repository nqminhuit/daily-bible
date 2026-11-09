package main

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/minh/daily-bible/internal/dateutil"
	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/lectionary"
	"github.com/minh/daily-bible/internal/parser"
	"github.com/minh/daily-bible/internal/reading"
	"github.com/minh/daily-bible/internal/scripture"
)

func databasePath() string {
	if path := os.Getenv("DB_PATH"); path != "" {
		return path
	}
	return constants.DBPath
}

func main() {
	var years string
	var window crawlWindow
	flag.StringVar(&years, "years", "", "comma-separated list of years (e.g. 2026,2027)")
	flag.StringVar(&window.from, "from", "", "first date to verify (YYYY-MM-DD; requires -to)")
	flag.StringVar(&window.to, "to", "", "last date to verify (YYYY-MM-DD; requires -from)")
	flag.BoolVar(&window.refresh, "refresh", false, "recheck already verified dates against the source")
	flag.BoolVar(&window.acceptCorrections, "accept-corrections", false, "allow reviewed source references to replace differing stored refs")
	flag.StringVar(&window.calendar, "calendar", reading.DefaultCalendar(), "calendar policy: vatican-vi, roman-general, or vietnam (explicit opt-in)")
	flag.StringVar(&window.calendarFile, "calendar-file", "", "import a versioned calendar manifest; does not approve reading mappings")
	flag.BoolVar(&window.offline, "offline", false, "resolve from local evidence only; never request source pages")
	flag.StringVar(&window.auditReport, "audit-report", "", "write an offline classification report; does not repair legacy refs")
	flag.StringVar(&window.reviewDate, "review-entry", "", "date identifying a reusable entry to explicitly review")
	flag.StringVar(&window.reviewRef, "entry-ref", "", "reviewed Gospel reference (not inferred from the old key)")
	flag.StringVar(&window.evidenceURL, "entry-evidence", "", "actual HTTPS evidence URL for the reviewed mapping")
	flag.StringVar(&window.evidenceDate, "evidence-date", "", "actual evidence date, if the source is a dated page")
	flag.StringVar(&window.reviewNote, "review-note", "", "explanation of the mapping's applicability and Mass option")
	flag.BoolVar(&window.replaceEntry, "replace-entry", false, "explicitly replace a previously reviewed mapping (retains history)")
	flag.StringVar(&window.acceptDate, "accept-date", "", "accept exact-date evidence for this calendar without approving its reusable entry")
	flag.StringVar(&window.dateLabel, "date-label", "", "reviewed celebration label for -accept-date")
	flag.BoolVar(&window.replaceDate, "replace-date", false, "explicitly replace a previously accepted date reference")
	flag.StringVar(&window.reviewRef, "ref", "", "reviewed Gospel reference for -review-entry or -accept-date")
	flag.StringVar(&window.evidenceURL, "evidence", "", "actual HTTPS evidence URL for the review")
	flag.Parse()
	if window.acceptDate != "" && window.from == "" && window.to == "" {
		window.from, window.to = window.acceptDate, window.acceptDate
	}
	if window.reviewDate != "" && window.from == "" && window.to == "" {
		window.from, window.to = window.reviewDate, window.reviewDate
	}

	var err error
	years, err = validateWindow(years, window)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err = run(ctx, years, window)
	stop()
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, years string, window crawlWindow) error {
	db, err := dbpkg.Open(databasePath())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	if err := dbpkg.EnsureReadingCache(ctx, db); err != nil {
		return fmt.Errorf("upgrade reading provenance/cache: %w", err)
	}
	if window.calendar == "" {
		window.calendar = reading.DefaultCalendar()
	}
	if window.calendar != reading.SourceCalendar && !lectionary.CalendarSupported(window.calendar) {
		return fmt.Errorf("unsupported calendar %q", window.calendar)
	}
	if err := prepareCalendar(ctx, db, window); err != nil {
		return err
	}
	if window.acceptDate != "" {
		return reading.ReviewDate(ctx, db, reading.EntryReview{CalendarID: window.calendar, Date: window.acceptDate, Ref: window.reviewRef, EvidenceURL: window.evidenceURL, EvidenceDate: window.evidenceDate, Note: window.reviewNote, Replace: window.replaceDate}, window.dateLabel)
	}
	if window.reviewDate != "" {
		return reading.ReviewEntry(ctx, db, reading.EntryReview{CalendarID: window.calendar, Date: window.reviewDate, Ref: window.reviewRef, EvidenceURL: window.evidenceURL, EvidenceDate: window.evidenceDate, Note: window.reviewNote, Replace: window.replaceEntry})
	}

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

	if window.auditReport != "" {
		return writeAudit(ctx, db, yearStrs, window)
	}
	jobWindow := window
	if window.calendar != reading.SourceCalendar || window.offline {
		jobWindow.refresh = true
	}
	jobs, err := pendingDates(ctx, db, yearStrs, jobWindow)
	if err != nil {
		return fmt.Errorf("query pending dates: %w", err)
	}
	log.Printf("found %d exact dates to verify", len(jobs))

	crawlClient := &http.Client{Timeout: 10 * time.Second}
	defer crawlClient.CloseIdleConnections()
	if window.calendar != reading.SourceCalendar || window.offline {
		stats, err := populateCacheFirst(ctx, db, crawlClient, jobs, window)
		log.Printf("cache summary: dates=%d cache_hits=%d http_requests=%d observation_candidates=%d unresolved=%d missing_text=%d source_differences=%d", stats.Dates, stats.CacheHits, stats.HTTPRequests, stats.Candidates, stats.Unresolved, stats.MissingText, stats.Differences)
		return err
	}
	return populateRefs(ctx, db, crawlClient, jobs, window.acceptCorrections)
}

func populateRefs(ctx context.Context, db *sql.DB, client *http.Client, jobs []dateKey, acceptCorrections bool) error {
	var failures []error
	for i, j := range jobs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := time.Parse("2006-01-02", j.date); err != nil {
			return fmt.Errorf("invalid job date %q: %w", j.date, err)
		}
		// Future dates may already be published. Never infer them from a key;
		// accept them only when the source page itself confirms this exact date.
		url := reading.SourceURL(j.date)

		ref, err := fetchGospelRef(ctx, client, url, j.date)
		if err != nil {
			log.Printf("[%d/%d] FAIL key=%s date=%s: %v", i+1, len(jobs), j.key, j.date, err)
			failures = append(failures, fmt.Errorf("verify %s: %w", j.date, err))
			if err := waitContext(ctx, 500*time.Millisecond); err != nil {
				return err
			}
			continue
		}

		if err := storeVerifiedReading(ctx, db, j.date, ref, url, acceptCorrections); err != nil {
			if !errors.Is(err, errReferenceConflict) {
				return fmt.Errorf("update %s: %w", j.date, err)
			}
			log.Printf("[%d/%d] CONFLICT: %v", i+1, len(jobs), err)
			failures = append(failures, err)
			if err := waitContext(ctx, 300*time.Millisecond); err != nil {
				return err
			}
			continue
		}

		log.Printf("[%d/%d] VERIFIED date=%s key=%s ref=%s source=%s", i+1, len(jobs), j.date, j.key, ref, url)
		if err := waitContext(ctx, 300*time.Millisecond); err != nil {
			return err
		}
	}
	return errors.Join(append(failures, ctx.Err())...)
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

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO daily_readings (date, lectionary_key) VALUES (?, ?)
		ON CONFLICT(date) DO UPDATE SET lectionary_key = excluded.lectionary_key
		WHERE daily_readings.lectionary_key <> excluded.lectionary_key`)
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
	return pendingDates(ctx, db, years, crawlWindow{})
}

func pendingDates(ctx context.Context, db *sql.DB, years []string, window crawlWindow) ([]dateKey, error) {
	// Build date range filters: date BETWEEN 'YYYY-01-01' AND 'YYYY-12-31'
	var conditions []string
	var args []interface{}
	for _, ys := range years {
		ys = strings.TrimSpace(ys)
		conditions = append(conditions, "dr.date BETWEEN ? AND ?")
		args = append(args, ys+"-01-01", ys+"-12-31")
	}
	dateFilter := "(" + strings.Join(conditions, " OR ") + ")"

	filter := dateFilter
	if window.from != "" {
		filter += " AND dr.date BETWEEN ? AND ?"
		args = append(args, window.from, window.to)
	} else {
		filter += " AND dr.date <= ?"
		args = append(args, dateutil.TodayDate())
	}
	if !window.refresh {
		filter += ` AND NOT EXISTS (
			SELECT 1 FROM daily_reading_sources s WHERE s.date = dr.date
			AND s.source_date = dr.date AND s.gospel_ref = dr.gospel_ref
			AND s.verified_at <> '' AND s.source_url =
			'https://www.vaticannews.va/vi/loi-chua-hang-ngay/' ||
			replace(dr.date, '-', '/') || '.html')`
	}
	query := fmt.Sprintf(`
		SELECT dr.lectionary_key, dr.date FROM daily_readings dr
		WHERE %s ORDER BY dr.date
	`, filter)

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
		if err != nil {
			return "", err
		}
		ref, err = scripture.NormalizeGospelRef(ref)
		if err != nil {
			return "", fmt.Errorf("invalid source reference: %w", err)
		}
		return ref, nil
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
