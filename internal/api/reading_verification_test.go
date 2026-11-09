package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minh/daily-bible/internal/dateutil"
)

func TestDateReadingRejectsUnverifiedAndStaleReferences(t *testing.T) {
	db := setupTestDB(t)
	today := dateutil.TodayDate()
	if _, err := db.Exec(`INSERT INTO verses(book,chapter,verse,text) VALUES ('Ga',2,13,'verified text');
		INSERT INTO daily_readings(date,lectionary_key,gospel_ref) VALUES (?, 'estimated_key', 'Ga 2,13')`, today); err != nil {
		t.Fatal(err)
	}
	h := makeTodayHandler(db)
	request := func(want int) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/today", nil))
		if w.Code != want {
			t.Fatalf("want %d, got %d: %s", want, w.Code, w.Body.String())
		}
		return w
	}
	request(http.StatusNotFound)
	verifyTestReading(t, db, today)
	w := request(http.StatusOK)
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["verified"] != true || result["source_url"] == "" || result["verified_at"] == "" {
		t.Fatalf("missing source evidence: %+v", result)
	}
	if result["celebration"] != "" {
		t.Fatal("estimated key must not become an authoritative celebration name")
	}
	if _, err := db.Exec(`UPDATE daily_readings SET gospel_ref='Lc 17,1-6' WHERE date=?`, today); err != nil {
		t.Fatal(err)
	}
	request(http.StatusNotFound)
}
