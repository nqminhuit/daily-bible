package api

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/minh/daily-bible/internal/constants"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// NewRouter initializes HTTP handlers and returns an http.Handler.
func NewRouter(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/liveness", makeLivenessHandler())
	mux.HandleFunc("/readiness", makeReadinessHandler(db, constants.DBPath))
	mux.HandleFunc("/api/v1/gospel/", makeGetGospelHandler(db))
	mux.HandleFunc("/api/v1/search", makeSearchHandler(db))
	mux.HandleFunc("/api/v1/random", makeRandomHandler(db))
	mux.HandleFunc("/api/v1/today", makeTodayHandler(db))
	mux.HandleFunc("/api/v1/date/", makeDateByPathHandler(db))
	mux.Handle("/", http.NotFoundHandler())
	return loggingMiddleware(corsMiddleware(mux))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
