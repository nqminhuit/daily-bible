package api

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/minh/daily-bible/internal/dateutil"
)

type failingResponseWriter struct {
	header          http.Header
	writes, headers int
}

func (w *failingResponseWriter) Header() http.Header { return w.header }
func (w *failingResponseWriter) WriteHeader(int)     { w.headers++ }
func (w *failingResponseWriter) Write(p []byte) (int, error) {
	w.writes++
	return len(p) / 2, errors.New("injected response write failure")
}

func TestHandlersLogJSONWriteFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO verses(book, chapter, verse, text) VALUES ('Ga', 1, 1, 'hello')`); err != nil {
		t.Fatal(err)
	}
	today := dateutil.TodayDate()
	if _, err := db.Exec(`INSERT INTO daily_readings(date, lectionary_key, gospel_ref) VALUES (?, 'key', 'Ga 1,1')`, today); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	for _, tc := range []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/api/v1/gospel/Ga%201,1", makeGetGospelHandler(db)},
		{"/api/v1/search?q=hello", makeSearchHandler(db)},
		{"/api/v1/random", makeRandomHandler(db)},
		{"/api/v1/today", makeTodayHandler(db)},
		{"/api/v1/date/" + today, makeDateByPathHandler(db)},
	} {
		t.Run(tc.path, func(t *testing.T) {
			logs.Reset()
			w := &failingResponseWriter{header: make(http.Header)}
			tc.handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.writes != 1 || w.headers != 0 {
				t.Fatalf("attempted second response: writes=%d headers=%d", w.writes, w.headers)
			}
			if !strings.Contains(logs.String(), "injected response write failure") {
				t.Fatalf("write failure not logged: %s", logs.String())
			}
			if w.header.Get("Content-Type") != "application/json" {
				t.Fatal("missing JSON content type")
			}
		})
	}
}

// A small SQL driver injects failures after QueryContext has successfully
// returned rows, so this covers rows.Err rather than only query startup errors.
type failingRowsDriver struct{}
type failingRowsConn struct{ scanFailure bool }
type failingSearchRows struct{ scanFailure, emitted bool }

func init() { sql.Register("api-failing-search-rows", failingRowsDriver{}) }
func (failingRowsDriver) Open(name string) (driver.Conn, error) {
	return &failingRowsConn{scanFailure: name == "scan"}, nil
}
func (*failingRowsConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (*failingRowsConn) Close() error              { return nil }
func (*failingRowsConn) Begin() (driver.Tx, error) { return nil, errors.New("unexpected transaction") }
func (c *failingRowsConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &failingSearchRows{scanFailure: c.scanFailure}, nil
}
func (*failingSearchRows) Columns() []string { return []string{"text"} }
func (*failingSearchRows) Close() error      { return nil }
func (r *failingSearchRows) Next(dest []driver.Value) error {
	if r.emitted {
		if r.scanFailure {
			return io.EOF
		}
		return errors.New("injected row iteration failure")
	}
	r.emitted = true
	if r.scanFailure {
		dest[0] = nil
	} else {
		dest[0] = "partial-result"
	}
	return nil
}

func TestSearchRowFailuresDoNotReturnPartialSuccess(t *testing.T) {
	for _, mode := range []string{"iteration", "scan"} {
		t.Run(mode, func(t *testing.T) {
			db, err := sql.Open("api-failing-search-rows", mode)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			w := httptest.NewRecorder()
			makeSearchHandler(db).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/search?q=hello", nil))
			if w.Code != http.StatusInternalServerError {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "partial-result") {
				t.Fatal("returned partial results")
			}
		})
	}
}
