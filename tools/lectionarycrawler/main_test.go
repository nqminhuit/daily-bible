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

func TestImportYear(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE daily_readings (
			date TEXT NOT NULL PRIMARY KEY,
			lectionary_key TEXT NOT NULL,
			gospel_ref TEXT
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	if err := importYear(db, 2026); err != nil {
		t.Fatalf("importYear(2026): %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM daily_readings`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 365 {
		t.Fatalf("expected 365 rows, got %d", count)
	}

	var key string
	err = db.QueryRow(`SELECT lectionary_key FROM daily_readings WHERE date = '2026-04-05'`).Scan(&key)
	if err != nil {
		t.Fatal(err)
	}
	if key != "easter_sunday_A" {
		t.Fatalf("expected easter_sunday_A, got %q", key)
	}
}

func TestMissingGospelRefs(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE daily_readings (
			date TEXT NOT NULL PRIMARY KEY,
			lectionary_key TEXT NOT NULL,
			gospel_ref TEXT
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		INSERT INTO daily_readings (date, lectionary_key, gospel_ref)
		VALUES
			('2026-01-26', 'ordinary_3_mon_II', NULL),
			('2026-01-27', 'ordinary_3_tue_II', NULL),
			('2026-03-22', 'lent_5_sun_A', 'Ga 11,1-45')
	`)
	if err != nil {
		t.Fatal(err)
	}

	jobs, err := missingGospelRefs(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(jobs) != 2 {
		t.Fatalf("expected 2 missing keys, got %d: %v", len(jobs), jobs)
	}

	expected := []string{"ordinary_3_mon_II", "ordinary_3_tue_II"}
	for i, j := range jobs {
		if j.key != expected[i] {
			t.Errorf("job[%d].key = %q, want %q", i, j.key, expected[i])
		}
	}
}

func TestParseDates(t *testing.T) {
	dates, err := parseDates("2026-08-04, 2026-08-06,2026-08-04")
	if err != nil {
		t.Fatalf("parseDates returned error: %v", err)
	}
	if len(dates) != 2 {
		t.Fatalf("expected 2 unique dates, got %d", len(dates))
	}
	if got := dates[0].Format("2006-01-02"); got != "2026-08-04" {
		t.Fatalf("dates[0] = %q, want %q", got, "2026-08-04")
	}
	if got := dates[1].Format("2006-01-02"); got != "2026-08-06" {
		t.Fatalf("dates[1] = %q, want %q", got, "2026-08-06")
	}
}

func TestParseDatesInvalid(t *testing.T) {
	if _, err := parseDates("08/04/2026"); err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestMissingGospelRefsForDates(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE daily_readings (
			date TEXT NOT NULL PRIMARY KEY,
			lectionary_key TEXT NOT NULL,
			gospel_ref TEXT
		);
		INSERT INTO daily_readings (date, lectionary_key, gospel_ref)
		VALUES
			('2026-08-04', 'ordinary_18_tue_II', NULL),
			('2026-08-05', 'ordinary_18_wed_II', 'Mt 15,21-28')
	`)
	if err != nil {
		t.Fatal(err)
	}

	dates, err := parseDates("2026-08-04,2026-08-05,2026-08-06")
	if err != nil {
		t.Fatal(err)
	}

	jobs, err := missingGospelRefsForDates(db, dates)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 missing date job, got %d: %v", len(jobs), jobs)
	}
	if jobs[0].date != "2026-08-04" || jobs[0].key != "ordinary_18_tue_II" {
		t.Fatalf("unexpected job: %+v", jobs[0])
	}
}
