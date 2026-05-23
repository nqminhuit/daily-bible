package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	migrationsDir := "../../data/migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 0 && e.Name()[0] >= '0' && e.Name()[0] <= '9' {
			content, err := os.ReadFile(migrationsDir + "/" + e.Name())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(string(content)); err != nil {
				t.Fatalf("apply migration %s: %v", e.Name(), err)
			}
		}
	}

	return db
}

func TestGetGospel(t *testing.T) {
	db := setupTestDB(t)

	if _, err := db.Exec(`
	INSERT INTO verses(book,chapter,verse,verse_suffix,text)
	VALUES('Ga',10,31,'','Jews picked up stones...')`); err != nil {
		t.Fatal(err)
	}

	handler := makeGetGospelHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/gospel/Ga%2010,31-31", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}
