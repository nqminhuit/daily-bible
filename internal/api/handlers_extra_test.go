package api_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/minh/daily-bible/internal/api"
	dbpkg "github.com/minh/daily-bible/internal/db"
)

func TestAPIValidation(t *testing.T) {
	tmpdb := filepath.Join(os.TempDir(), "daily_bible_api_validation.db")
	defer os.Remove(tmpdb)
	os.Remove(tmpdb)

	dbConn, err := dbpkg.Open(tmpdb)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer dbConn.Close()

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	schemaPath := filepath.Join(root, "data", "schema.sql")
	if err := dbpkg.InitDB(dbConn, schemaPath); err != nil {
		t.Fatalf("init db: %v", err)
	}

	_, err = dbConn.Exec(`INSERT INTO gospels(reference, book, chapter_start, verse_start, chapter_end, verse_end, text) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"Mt 1,1-2", "Mt", 1, 1, 1, 2, "Jesus sample")
	if err != nil {
		t.Fatalf("insert sample: %v", err)
	}

	router := api.NewRouter(dbConn)
	ts := httptest.NewServer(router)
	defer ts.Close()

	// empty q
	resp, err := http.Get(ts.URL + "/api/v1/search?q=")
	if err != nil {
		t.Fatalf("search empty q error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty q, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// too long q
	longQ := strings.Repeat("x", 201)
	resp, err = http.Get(ts.URL + "/api/v1/search?q=" + url.QueryEscape(longQ))
	if err != nil {
		t.Fatalf("search long q error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for long q, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// empty ref
	resp, err = http.Get(ts.URL + "/api/v1/gospel/")
	if err != nil {
		t.Fatalf("gospel empty ref error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty ref, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// nonexistent -> 404
	resp, err = http.Get(ts.URL + "/api/v1/gospel/" + url.PathEscape("Nope 1,1"))
	if err != nil {
		t.Fatalf("gospel nonexistent: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent ref, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}
