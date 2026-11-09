package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dbpkg "github.com/minh/daily-bible/internal/db"
)

func newCrawlerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if err := dbpkg.EnsureReadingSources(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

const readingsSchema = `CREATE TABLE daily_readings(date TEXT PRIMARY KEY, lectionary_key TEXT NOT NULL, gospel_ref TEXT)`

func TestImportYearRollback(t *testing.T) {
	for _, failure := range []string{"prepare", "insert", "commit"} {
		t.Run(failure, func(t *testing.T) {
			db := newCrawlerTestDB(t)
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			switch failure {
			case "insert":
				if _, err := db.ExecContext(ctx, readingsSchema+`; CREATE TRIGGER reject_insert BEFORE INSERT ON daily_readings WHEN (SELECT COUNT(*) FROM daily_readings) >= 5 BEGIN SELECT RAISE(ABORT, 'injected insert failure'); END;`); err != nil {
					t.Fatal(err)
				}
			case "commit":
				if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON; CREATE TABLE keys(key TEXT PRIMARY KEY); CREATE TABLE daily_readings(date TEXT PRIMARY KEY, lectionary_key TEXT REFERENCES keys(key) DEFERRABLE INITIALLY DEFERRED, gospel_ref TEXT)`); err != nil {
					t.Fatal(err)
				}
			}
			if err := importYear(ctx, db, 2026); err == nil || !strings.Contains(err.Error(), failure) {
				t.Fatalf("expected %s failure, got %v", failure, err)
			}
			if failure == "prepare" {
				if _, err := db.ExecContext(ctx, readingsSchema); err != nil {
					t.Fatalf("connection not released: %v", err)
				}
			}
			var count int
			if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM daily_readings`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("failed transaction left %d rows", count)
			}
			if failure == "insert" {
				if _, err := db.ExecContext(ctx, `DROP TRIGGER reject_insert`); err != nil {
					t.Fatal(err)
				}
			}
			if failure != "commit" {
				if err := importYear(ctx, db, 2026); err != nil {
					t.Fatalf("database not usable after rollback: %v", err)
				}
			}
		})
	}
}

func TestCancelledDatabaseOperations(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := importYear(ctx, db, 2026); !errors.Is(err, context.Canceled) {
		t.Fatalf("import: %v", err)
	}
	if _, err := missingGospelRefs(ctx, db, []string{"2026"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("query: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM daily_readings`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cancelled import wrote %d rows", count)
	}
}

func TestWaitContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- waitContext(ctx, time.Hour) }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("wait ignored cancellation")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchCancellationStopsRetries(t *testing.T) {
	for _, duringRequest := range []bool{false, true} {
		name := "backoff"
		if duringRequest {
			name = "request"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var calls atomic.Int32
			started := make(chan struct{})
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				close(started)
				if duringRequest {
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("error")), Header: make(http.Header)}, nil
			})}
			done := make(chan error, 1)
			go func() { _, err := fetchGospelRef(ctx, client, "https://example.test/", "2026-01-01"); done <- err }()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("request did not start")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("got %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("fetch ignored cancellation")
			}
			if calls.Load() != 1 {
				t.Fatalf("made %d requests after cancellation", calls.Load())
			}
		})
	}
}

func TestPopulateCancellationStopsJobs(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		close(started)
		<-r.Context().Done()
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	// Route Vatican requests to the local server while retaining their context.
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, srv.URL, nil)
		if err != nil {
			return nil, err
		}
		return srv.Client().Transport.RoundTrip(req)
	})}
	done := make(chan error, 1)
	go func() {
		done <- populateRefs(ctx, db, client, []dateKey{{date: "2026-01-01", key: "one"}, {date: "2026-01-02", key: "two"}}, false)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("crawler did not stop")
	}
	if requests.Load() != 1 {
		t.Fatalf("processed subsequent jobs: %d requests", requests.Load())
	}
}
