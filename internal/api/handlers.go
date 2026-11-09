package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/dateutil"
	"github.com/minh/daily-bible/internal/query"
	"github.com/minh/daily-bible/internal/reading"
)

// writeJSON logs output errors without attempting a second response: Encode
// may already have written headers or part of the body before failing.
func writeJSON(w http.ResponseWriter, r *http.Request, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("JSON response error for %s %s: %v", r.Method, r.URL.Path, err)
	}
}

func makeGetGospelHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := "/api/v1/gospel/"
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		encRef := strings.TrimPrefix(r.URL.Path, prefix)
		if encRef == "" {
			http.Error(w, "reference required", http.StatusBadRequest)
			return
		}
		ref, err := url.PathUnescape(encRef)
		if err != nil {
			http.Error(w, "invalid reference", http.StatusBadRequest)
			return
		}

		parsed, err := ParseRefMultiChapter(ref)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid reference: %v", err), http.StatusBadRequest)
			return
		}

		verses, err := queryParsedRef(r.Context(), db, parsed)
		if err != nil {
			log.Printf("gospel query error for ref %q: %v", ref, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if len(verses) == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		writeJSON(w, r, verses)
	}
}

func makeSearchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if strings.TrimSpace(q) == "" {
			http.Error(w, "q query required", http.StatusBadRequest)
			return
		}
		if len(q) > 200 {
			http.Error(w, "query too long", http.StatusBadRequest)
			return
		}
		rows, err := db.QueryContext(r.Context(), `SELECT text FROM verses_fts WHERE verses_fts MATCH ? LIMIT 10`,
			query.FtsPhraseQuery(q))
		if err != nil {
			log.Printf("fts search error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var results []string
		for rows.Next() {
			var text string
			if err := rows.Scan(&text); err != nil {
				log.Printf("scan error: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			results = append(results, text)
		}
		if err := rows.Err(); err != nil {
			log.Printf("fts search rows error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, r, results)
	}
}

// maxRandomRetries is the number of random rowid attempts before giving up.
// The retry loop handles rowid gaps caused by deleted rows; 10 retries
// provides sufficient probability of finding an existing row even with
// moderate deletion density.
const maxRandomRetries = 10

// makeRandomHandler returns a handler that serves a random verse from the database.
// MAX(rowid) is queried per-request so the handler works correctly even if
// rows are inserted after server startup.
func makeRandomHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var maxRowID int64
		if err := db.QueryRowContext(r.Context(), "SELECT IFNULL(MAX(rowid), 0) FROM verses").Scan(&maxRowID); err != nil {
			log.Printf("query max rowid: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if maxRowID <= 0 {
			http.Error(w, "no verses available", http.StatusNotFound)
			return
		}
		var text string
		for range maxRandomRetries {
			randomID := 1 + rand.Int64N(maxRowID)
			row := db.QueryRowContext(r.Context(), "SELECT text FROM verses WHERE rowid = ?", randomID)
			if err := row.Scan(&text); err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				log.Printf("random query error: %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			writeJSON(w, r, text)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func makeTodayHandler(db *sql.DB) http.HandlerFunc {
	return makeDateHandler(db, func() string {
		return dateutil.TodayDate()
	})
}

func makeDateByPathHandler(db *sql.DB) http.HandlerFunc {
	return makeDateHandler(db, func() string {
		return ""
	})
}

func makeDateHandler(db *sql.DB, getDate func() string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := getDate()
		if date == "" {
			prefix := "/api/v1/date/"
			if !strings.HasPrefix(r.URL.Path, prefix) {
				http.NotFound(w, r)
				return
			}
			date = strings.TrimPrefix(r.URL.Path, prefix)
			if date == "" {
				http.Error(w, "date required", http.StatusBadRequest)
				return
			}
		}

		if _, err := time.Parse("2006-01-02", date); err != nil {
			http.Error(w, "invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		metadata, err := reading.Lookup(r.Context(), db, date)
		gospelRef := metadata.Ref

		if err == sql.ErrNoRows {
			http.Error(w, "no verified reading found for date", http.StatusNotFound)
			return
		}
		if err != nil {
			log.Printf("today query error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		verses, err := QueryByRef(r.Context(), db, gospelRef)
		if err != nil {
			log.Printf("verses query error for ref %q: %v", gospelRef, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if len(verses) == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var sb strings.Builder
		for _, v := range verses {
			sb.WriteString(v.Text)
			sb.WriteByte(' ')
		}
		combined := strings.TrimSpace(sb.String())

		writeJSON(w, r, map[string]any{
			"date":           date,
			"lectionary_key": metadata.LectionaryKey,
			"ref":            gospelRef,
			"verses":         combined,
			"verified":       true,
			"source_url":     metadata.SourceURL,
			"verified_at":    metadata.VerifiedAt,
			"celebration":    metadata.Celebration,
		})
	}
}
