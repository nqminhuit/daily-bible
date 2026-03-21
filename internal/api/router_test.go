package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestNewRouter_RegistersHandlers_404(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r, err := NewRouter(db)
	if err != nil {
		t.Fatal(err)
	}

	// no rows -> 404
	req := httptest.NewRequest("GET", "/api/v1/random", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when no rows, got %d", w.Code)
	}
}

func TestNewRouter_RegistersHandlers_Ok(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// insert row and expect 200
	if _, err := db.Exec(`INSERT INTO verses(book,chapter,verse,text) VALUES('Ga',10,31,'text from router')`); err != nil {
		t.Fatal(err)
	}

	r, err := NewRouter(db)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/v1/random", nil)
	w := httptest.NewRecorder()
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after insert, got %d", w.Code)
	}
}
