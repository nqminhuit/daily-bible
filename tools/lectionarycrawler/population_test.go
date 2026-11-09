package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

const gospelPage = `<html><head><title>Tin Mừng và Lời Chúa ngày 05 tháng 4 2026 - Vatican News</title></head><body><section><div class="section__content"><p>✠Bài trích Tin Mừng Chúa Giê-su Ki-tô theo thánh Gio-an. Ga 11,1-45</p><p><sup>1</sup>Verse one</p></div></section></body></html>`

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b cancelOnClose) Close() error { b.cancel(); return b.ReadCloser.Close() }

func TestPopulateUpdateCancellation(t *testing.T) {
	db := newCrawlerTestDB(t)
	if _, err := db.Exec(readingsSchema + `; INSERT INTO daily_readings(date, lectionary_key) VALUES ('2026-04-05', 'key')`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: cancelOnClose{io.NopCloser(strings.NewReader(gospelPage)), cancel}, Header: make(http.Header)}, nil
	})}
	// Cancel after the body is read, before the database update can succeed.
	err := populateRefs(ctx, db, client, []dateKey{{date: "2026-04-05", key: "key"}}, "2026-12-31")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	var ref sql.NullString
	if err := db.QueryRow(`SELECT gospel_ref FROM daily_readings`).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	if ref.Valid {
		t.Fatalf("cancelled crawl updated database: %q", ref.String)
	}
}

func TestPopulateUpdateFailure(t *testing.T) {
	db := newCrawlerTestDB(t) // No table: updating must fail, not log and continue.
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(gospelPage)), Header: make(http.Header)}, nil
	})}
	err := populateRefs(t.Context(), db, client, []dateKey{{date: "2026-04-05", key: "key"}}, "2026-12-31")
	if err == nil || !strings.Contains(err.Error(), "update key:") {
		t.Fatalf("expected update error, got %v", err)
	}
}
