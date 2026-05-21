package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestFetchGospelRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html><body>
			<section>
			<div class="section__content">
			<p>✠Bài trích Tin Mừng Chúa Giê-su Ki-tô theo thánh Gio-an. Ga 11,1-45</p>
			<p><sup>1</sup>Verse one text.</p>
			<p><sup>2</sup>Verse two text.</p>
			</div>
			</section>
			</body></html>
		`))
	}))
	defer server.Close()

	client := &http.Client{}
	ref, err := fetchGospelRef(client, server.URL)
	if err != nil {
		t.Fatalf("fetchGospelRef error: %v", err)
	}
	if ref != "Ga 11,1-45" {
		t.Fatalf("expected Ga 11,1-45, got %q", ref)
	}
}

func TestFetchGospelRefNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &http.Client{}
	_, err := fetchGospelRef(client, server.URL)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestFetchGospelRefConnectionError(t *testing.T) {
	client := &http.Client{}
	_, err := fetchGospelRef(client, "http://localhost:1")
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestFetchGospelRefMissingRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html><body>
			<section>
			<div class="section__content">
			<p>No bible reference here</p>
			<p><sup>1</sup>Some text.</p>
			</div>
			</section>
			</body></html>
		`))
	}))
	defer server.Close()

	client := &http.Client{}
	_, err := fetchGospelRef(client, server.URL)
	if err == nil {
		t.Fatal("expected error when no gospel ref found")
	}
}

func TestFetchGospelRefNonContiguous(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
			<html><body>
			<section>
			<div class="section__content">
			<p>✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Mát-thêu. Mt 5,20-22a.27-28</p>
			<p><sup>20</sup>Verse twenty.</p>
			<p><sup>21</sup>Verse twenty-one.</p>
			</div>
			</section>
			</body></html>
		`))
	}))
	defer server.Close()

	client := &http.Client{}
	ref, err := fetchGospelRef(client, server.URL)
	if err != nil {
		t.Fatalf("fetchGospelRef error: %v", err)
	}
	if ref != "Mt 5,20-22a.27-28" {
		t.Fatalf("expected Mt 5,20-22a.27-28, got %q", ref)
	}
}

func TestMissingKeysQuery(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE calendar (
			date TEXT NOT NULL PRIMARY KEY,
			lectionary_key TEXT NOT NULL,
			season TEXT NOT NULL,
			sunday_cycle TEXT NOT NULL,
			weekday TEXT NOT NULL,
			weekday_cycle TEXT NOT NULL,
			week_of_season INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE lectionary (
			lectionary_key TEXT NOT NULL PRIMARY KEY,
			gospel_ref TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		INSERT INTO calendar (date, lectionary_key, season, sunday_cycle, weekday, weekday_cycle, week_of_season)
		VALUES
			('2026-01-26', 'ordinary_3_mon_II', 'ordinary', '', 'mon', 'II', 3),
			('2026-01-27', 'ordinary_3_tue_II', 'ordinary', '', 'tue', 'II', 3),
			('2026-03-22', 'lent_5_sun_A', 'lent', 'A', 'sun', '', 5)
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		INSERT INTO lectionary (lectionary_key, gospel_ref)
		VALUES ('lent_5_sun_A', 'Ga 11,1-45')
	`)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query(`
		SELECT DISTINCT c.lectionary_key, c.date
		FROM calendar c
		LEFT JOIN lectionary l ON c.lectionary_key = l.lectionary_key
		WHERE l.lectionary_key IS NULL
		GROUP BY c.lectionary_key
		ORDER BY c.date
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key, date string
		if err := rows.Scan(&key, &date); err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 missing keys, got %d: %v", len(keys), keys)
	}

	expected := []string{"ordinary_3_mon_II", "ordinary_3_tue_II"}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("key[%d]: got %q, want %q", i, k, expected[i])
		}
	}
}
