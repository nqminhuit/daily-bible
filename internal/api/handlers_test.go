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
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	schema, err := os.ReadFile("../../data/schema.sql")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(string(schema))
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestGetGospel(t *testing.T) {
	db := setupTestDB(t)

	if _, err := db.Exec(`
	INSERT INTO verses(book,chapter,verse,text)
	VALUES('Ga',10,31,'Jews picked up stones...')`); err != nil {
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
