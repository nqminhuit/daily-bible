package api

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"
)

func makeLivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func makeReadinessHandler(db *sql.DB, dbPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(dbPath); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "database not ready", http.StatusServiceUnavailable)
				return
			}
			http.Error(w, "readiness check failed", http.StatusInternalServerError)
			return
		}
		if db == nil {
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()

		var one int
		err := db.QueryRowContext(ctx, "SELECT 1 FROM verses LIMIT 1").Scan(&one)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("readiness db probe error: %v", err)
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
