package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/minh/daily-bible/internal/constants"
)

// NewRouter initializes HTTP handlers and returns an http.Handler. Returns an error
// if required startup checks (like reading the max rowid) fail so the caller can
// handle startup failures gracefully.
func NewRouter(db *sql.DB) (http.Handler, error) {
	var maxRowID int64
	if err := db.QueryRow("SELECT IFNULL(MAX(rowid), 0) FROM verses").Scan(&maxRowID); err != nil {
		return nil, fmt.Errorf("query max rowid: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/liveness", makeLivenessHandler())
	mux.HandleFunc("/readiness", makeReadinessHandler(constants.DBPath))
	mux.HandleFunc("/api/v1/gospel/", makeGetGospelHandler(db))
	mux.HandleFunc("/api/v1/search", makeSearchHandler(db))
	mux.HandleFunc("/api/v1/random", makeRandomHandler(db, maxRowID))
	return mux, nil
}
