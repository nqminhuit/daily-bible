package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minh/daily-bible/internal/dateutil"
)

func TestDateReadingWithoutMatchingVerses(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	today := dateutil.TodayDate()
	if _, err := db.Exec(`INSERT INTO daily_readings (date, lectionary_key, gospel_ref) VALUES (?, 'test_key', 'Ga 11,1-2')`, today); err != nil {
		t.Fatal(err)
	}
	verifyTestReading(t, db, today)

	for _, tc := range []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{"today", "/api/v1/today", makeTodayHandler(db)},
		{"date", "/api/v1/date/" + today, makeDateByPathHandler(db)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != http.StatusNotFound {
				t.Fatalf("expected 404 for reading without matching verses, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
