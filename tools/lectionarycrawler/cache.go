package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/reading"
	"github.com/minh/daily-bible/internal/scripture"
)

func prepareCalendar(ctx context.Context, db *sql.DB, window crawlWindow) error {
	if window.calendarFile == "" {
		return nil
	}
	f, err := os.Open(window.calendarFile)
	if err != nil {
		return err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	if stat.Size() > 32<<20 {
		return fmt.Errorf("calendar manifest exceeds 32 MiB")
	}
	var manifest reading.CalendarManifest
	decoder := json.NewDecoder(io.LimitReader(f, 32<<20))
	if err := decoder.Decode(&manifest); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("calendar manifest has trailing data")
	}
	if manifest.CalendarID != window.calendar {
		return fmt.Errorf("manifest calendar %q differs from selected calendar %q", manifest.CalendarID, window.calendar)
	}
	return reading.ImportCalendar(ctx, db, manifest)
}

func writeAudit(ctx context.Context, db *sql.DB, years []string, w crawlWindow) error {
	report := reading.AuditReport{CalendarID: w.calendar, Counts: map[string]int{}}
	appendRange := func(from, to string) error {
		r, err := reading.Audit(ctx, db, w.calendar, from, to)
		if err != nil {
			return err
		}
		report.Rows = append(report.Rows, r.Rows...)
		report.SourceDifferences += r.SourceDifferences
		for status, n := range r.Counts {
			report.Counts[status] += n
		}
		return nil
	}
	if w.from != "" {
		if err := appendRange(w.from, w.to); err != nil {
			return err
		}
	} else {
		seen := map[string]bool{}
		for _, year := range years {
			year = strings.TrimSpace(year)
			if seen[year] {
				continue
			}
			seen[year] = true
			if err := appendRange(year+"-01-01", year+"-12-31"); err != nil {
				return err
			}
		}
	}
	if err := reading.FinalizeAudit(ctx, db, &report); err != nil {
		return err
	}
	f, err := os.Create(w.auditReport)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(report)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	log.Printf("offline audit: calendar=%s dates=%d entries=%d http_requests=0 counts=%v", w.calendar, len(report.Rows), len(report.Entries), report.Counts)
	return closeErr
}

type cacheStats struct{ Dates, CacheHits, HTTPRequests, Candidates, Unresolved, MissingText, Differences int }
type countingTransport struct {
	base  http.RoundTripper
	count *int
}

func (t countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	*t.count = *t.count + 1
	return t.base.RoundTrip(r)
}

// A missing mapping may yield one observation candidate per actual entry, but
// is never silently promoted to a reusable rule. Warm-cache calls do no HTTP.
func populateCacheFirst(ctx context.Context, db *sql.DB, client *http.Client, jobs []dateKey, w crawlWindow) (cacheStats, error) {
	stats := cacheStats{Dates: len(jobs)}
	copyClient := *client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	copyClient.Transport = countingTransport{base: base, count: &stats.HTTPRequests}
	seen := map[string]bool{}
	var failures []error
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		m, err := reading.LookupFor(ctx, db, job.date, w.calendar, reading.DayMass)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return stats, err
		}
		trusted := err == nil
		if trusted {
			complete, err := scripture.HasText(ctx, db, m.Ref)
			if err != nil {
				return stats, err
			}
			if !complete {
				stats.MissingText++
				log.Printf("MISSING TEXT date=%s ref=%s; repair the shared verse store, not the reference cache", job.date, m.Ref)
				continue
			}
			stats.CacheHits++
			if !w.refresh {
				continue
			}
		}
		if !trusted {
			stats.Unresolved++
		}
		if w.offline {
			continue
		}
		if w.calendar == reading.SourceCalendar {
			return stats, fmt.Errorf("source-only cache-first crawling must use populateRefs")
		}
		decided, err := reading.HasDateDecision(ctx, db, w.calendar, job.date, reading.DayMass)
		if err != nil {
			return stats, err
		}
		if decided && !trusted {
			log.Printf("DATE REVIEW REQUIRED date=%s calendar=%s; no automatic fallback/crawl", job.date, w.calendar)
			continue
		}
		key, version := m.EntryKey, m.RulesVersion
		if !trusted {
			key, version, err = reading.EntryForDate(ctx, db, w.calendar, job.date)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return stats, err
			}
		}
		if !trusted && w.calendar == "vietnam" {
			definition, err := reading.CalendarDefinition(ctx, db, w.calendar, job.date, version)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return stats, err
			}
			if err == nil && definition.Mass != nil {
				log.Printf("FORMULARY REVIEW REQUIRED date=%s mass=%s; dated Vatican reference is not national formulary evidence", job.date, definition.Mass.Formulary)
				continue
			}
		}
		if !w.refresh {
			if seen[key] {
				continue
			}
			seen[key] = true
			candidate, err := reading.HasObservationCandidate(ctx, db, w.calendar, key, version)
			if err != nil {
				return stats, err
			}
			if candidate {
				stats.Candidates++
				continue
			}
		}
		ref, err := fetchGospelRef(ctx, &copyClient, reading.SourceURL(job.date), job.date)
		if err != nil {
			failures = append(failures, fmt.Errorf("observe %s: %w", job.date, err))
			if err := waitContext(ctx, 500*time.Millisecond); err != nil {
				return stats, err
			}
			continue
		}
		if err := reading.ObserveVatican(ctx, db, job.date, ref, reading.SourceURL(job.date)); err != nil {
			return stats, err
		}
		stats.Candidates++
		log.Printf("OBSERVED date=%s entry=%s ref=%s; review scope and Mass option before -review-entry", job.date, key, ref)
		// A refresh audits a trusted mapping; an observation of another option or
		// calendar must not poison it. Report the difference for review instead.
		if trusted && !scripture.Equivalent(m.Ref, ref) {
			stats.Differences++
			failures = append(failures, fmt.Errorf("source/calendar difference date=%s calendar=%s accepted=%s observed=%s; reviewed selection retained", job.date, w.calendar, m.Ref, ref))
		}
		if err := waitContext(ctx, 300*time.Millisecond); err != nil {
			return stats, err
		}
	}
	if stats.Unresolved > 0 || stats.MissingText > 0 {
		failures = append(failures, fmt.Errorf("calendar=%s unresolved=%d missing_text=%d (observations are candidates, not approved mappings)", w.calendar, stats.Unresolved, stats.MissingText))
	}
	return stats, errors.Join(append(failures, ctx.Err())...)
}
