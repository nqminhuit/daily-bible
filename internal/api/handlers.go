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
)

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

		if _, _, _, err := ParseRef(ref); err != nil {
			http.Error(w, fmt.Sprintf("invalid reference: %v", err), http.StatusBadRequest)
			return
		}

		verses, err := QueryByRef(r.Context(), db, ref)
		if err != nil {
			log.Printf("gospel query error for ref %q: %v", ref, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if len(verses) == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(verses)
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
		if db == nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
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
			var ref string
			if err := rows.Scan(&ref); err != nil {
				log.Printf("scan error: %v", err)
				continue
			}
			results = append(results, ref)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
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
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(text)
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

		var lectionaryKey, gospelRef string
		err := db.QueryRowContext(r.Context(), `
			SELECT lectionary_key, gospel_ref
			FROM daily_readings
			WHERE date = ? AND gospel_ref IS NOT NULL
		`, date).Scan(&lectionaryKey, &gospelRef)

		if err == sql.ErrNoRows {
			http.Error(w, "no reading found for date", http.StatusNotFound)
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

		var sb strings.Builder
		for _, v := range verses {
			sb.WriteString(v.Text)
			sb.WriteByte(' ')
		}
		combined := strings.TrimSpace(sb.String())

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"date":           date,
			"lectionary_key": lectionaryKey,
			"ref":            gospelRef,
			"verses":         combined,
		})
	}
}
