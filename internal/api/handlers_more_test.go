package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/minh/daily-bible/internal/model"
)

func TestLivenessHandler(t *testing.T) {
	h := makeLivenessHandler()
	req := httptest.NewRequest("GET", "/liveness", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for liveness, got %d", w.Code)
	}
}

func TestReadinessHandler(t *testing.T) {
	t.Run("ready when db file exists", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		dir := t.TempDir()
		dbPath := filepath.Join(dir, "bible.db")
		if err := os.WriteFile(dbPath, []byte("ready"), 0o644); err != nil {
			t.Fatal(err)
		}

		h := makeReadinessHandler(db, dbPath)
		req := httptest.NewRequest("GET", "/readiness", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 when db file exists, got %d", w.Code)
		}
	})

	t.Run("not ready when db file is missing", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		h := makeReadinessHandler(db, filepath.Join(t.TempDir(), "missing.db"))
		req := httptest.NewRequest("GET", "/readiness", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 when db file is missing, got %d", w.Code)
		}
	})

	t.Run("not ready when db query fails", func(t *testing.T) {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		dir := t.TempDir()
		dbPath := filepath.Join(dir, "bible.db")
		if err := os.WriteFile(dbPath, []byte("ready"), 0o644); err != nil {
			t.Fatal(err)
		}

		h := makeReadinessHandler(db, dbPath)
		req := httptest.NewRequest("GET", "/readiness", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 when db probe fails, got %d", w.Code)
		}
	})
}

func TestGetGospel_ValidAndErrors(t *testing.T) {
	// prefix not found (no DB usage required)
	h := makeGetGospelHandler(nil)
	req := httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/foo"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-prefix path, got %d", w.Code)
	}

	// reference required
	db := setupTestDB(t)
	defer db.Close()
	h = makeGetGospelHandler(db)
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty ref, got %d", w.Code)
	}

	// invalid escape
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/%"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid escape, got %d", w.Code)
	}

	// invalid reference format (no space)
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/Ga"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid ref format, got %d", w.Code)
	}

	// invalid chapter part (missing comma)
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/Ga%2010"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing chapter comma, got %d", w.Code)
	}

	// invalid chapter number
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/Ga%20x,31-31"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid chapter, got %d", w.Code)
	}

	// single verse (no dash) is now valid, will return 404 when no rows match
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/Ga%2010,31"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for single verse with no matching rows, got %d", w.Code)
	}

	// invalid verse numbers
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/Ga%2010,xx-yy"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid verse numbers, got %d", w.Code)
	}

	// DB error (closed DB)
	dbErr := setupTestDB(t)
	dbErr.Close()
	h = makeGetGospelHandler(dbErr)
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/Ga%2010,31-31"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for db error, got %d", w.Code)
	}

	// not found (no rows)
	dbEmpty := setupTestDB(t)
	defer dbEmpty.Close()
	h = makeGetGospelHandler(dbEmpty)
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/Ga%2010,31-31"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when no rows, got %d", w.Code)
	}

	// success
	dbOK := setupTestDB(t)
	defer dbOK.Close()
	if _, err := dbOK.Exec(`INSERT INTO verses(book,chapter,verse,verse_suffix,text) VALUES('Ga',10,31,'','Jews picked up stones...')`); err != nil {
		t.Fatal(err)
	}
	h = makeGetGospelHandler(dbOK)
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/Ga%2010,31-31"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content-type, got %q", ct)
	}
	var got []model.Gospel
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != "Jews picked up stones..." {
		t.Fatalf("unexpected payload: %v", got)
	}

	// valid format with extra spaces between book/chapter parts
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/gospel/Ga%20%2010,31-31"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for ref with extra spaces, got %d", w.Code)
	}
}

func TestSearchHandler_ValidationAndDB(t *testing.T) {
	h := makeSearchHandler(nil)
	// q missing
	req := httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/search"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing q, got %d", w.Code)
	}

	// q too long
	long := strings.Repeat("a", 201)
	req = httptest.NewRequest("GET", "/?q=", nil)
	req.URL.Path = "/api/v1/search"
	req.URL.RawQuery = "q=" + url.QueryEscape(long)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long q, got %d", w.Code)
	}

	// DB error
	dbErr := setupTestDB(t)
	dbErr.Close()
	h = makeSearchHandler(dbErr)
	req = httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/search"
	req.URL.RawQuery = "q=hello"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for db error, got %d", w.Code)
	}
}

func TestSearchHandler_SuccessFTS(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	if _, err := db.Exec("INSERT INTO verses(book, chapter, verse, text) VALUES('Ga', 1, 1, 'hello world')"); err != nil {
		t.Fatal(err)
	}

	h := makeSearchHandler(db)
	req := httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/search"
	req.URL.RawQuery = "q=" + url.QueryEscape("hello world")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for fts search, got %d", w.Code)
	}
	var got []string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatalf("expected at least one search result, got empty array")
	}
}

func TestSearchHandler_QueryWithQuotes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	if _, err := db.Exec("INSERT INTO verses(book, chapter, verse, text) VALUES('Ga', 1, 1, ?)", `he said "hello"`); err != nil {
		t.Fatal(err)
	}

	h := makeSearchHandler(db)
	req := httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/search"
	req.URL.RawQuery = "q=" + url.QueryEscape(`he said "hello"`)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for query containing quotes, got %d", w.Code)
	}
}

// Test that search returns an empty JSON array when FTS exists but there are no matches.
func TestSearchHandler_EmptyResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	h := makeSearchHandler(db)
	req := httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/search"
	req.URL.RawQuery = "q=" + url.QueryEscape("no match here")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty fts search, got %d", w.Code)
	}
	var got []string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty array for no matches, got %v", got)
	}
}

func TestRandomHandler_Behavior(t *testing.T) {
	// no rows -> 404
	db := setupTestDB(t)
	defer db.Close()
	h := makeRandomHandler(db, 1)
	req := httptest.NewRequest("GET", "/", nil)
	req.URL.Path = "/api/v1/random"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when no rows, got %d", w.Code)
	}

	// success
	if _, err := db.Exec(`INSERT INTO verses(book,chapter,verse,verse_suffix,text) VALUES('Ga',10,31,'','random text')`); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for random success, got %d", w.Code)
	}
	var got string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got != "random text" {
		t.Fatalf("unexpected random payload: %q", got)
	}

	// db error
	dbErr := setupTestDB(t)
	dbErr.Close()
	h = makeRandomHandler(dbErr, 1)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for closed db, got %d", w.Code)
	}
}
