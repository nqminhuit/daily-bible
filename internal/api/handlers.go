package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/minh/daily-bible/internal/model"
)

func makeLivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func makeReadinessHandler(dbPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(dbPath); err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "database not ready", http.StatusServiceUnavailable)
				return
			}
			http.Error(w, "readiness check failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
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

		// Expected: "Ga 10,31-42"
		parts := strings.Split(ref, " ")
		if len(parts) != 2 {
			http.Error(w, "invalid reference format", http.StatusBadRequest)
			return
		}

		book := parts[0]

		chParts := strings.Split(parts[1], ",")
		if len(chParts) != 2 {
			http.Error(w, "invalid reference format", http.StatusBadRequest)
			return
		}

		chapter, err := strconv.Atoi(chParts[0])
		if err != nil {
			http.Error(w, "invalid chapter", http.StatusBadRequest)
			return
		}

		rangeParts := strings.Split(chParts[1], "-")
		if len(rangeParts) != 2 {
			http.Error(w, "invalid verse range", http.StatusBadRequest)
			return
		}

		vStart, err := strconv.Atoi(rangeParts[0])
		if err != nil {
			http.Error(w, "invalid verse", http.StatusBadRequest)
			return
		}

		vEnd, err := strconv.Atoi(rangeParts[1])
		if err != nil {
			http.Error(w, "invalid verse", http.StatusBadRequest)
			return
		}

		rows, err := db.Query(`
			SELECT book, chapter, verse, text
			FROM verses
			WHERE book = ?
			AND chapter = ?
			AND verse BETWEEN ? AND ?
			ORDER BY verse`,
			book, chapter, vStart, vEnd,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var results []model.Gospel
		for rows.Next() {
			var g model.Gospel
			if err := rows.Scan(&g.Book, &g.Chapter, &g.Verse, &g.Text); err != nil {
				http.Error(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
				return
			}
			results = append(results, g)
		}

		if len(results) == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
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

		rows, err := db.Query(`SELECT text FROM verses_fts WHERE verses_fts MATCH ? LIMIT 10`,
			fmt.Sprintf(`"%s"`, q))
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

// makeRandomHandler returns a handler that serves a random verse from the database.
// The table "verses" is expected to be static, immutable,
// and have a rowid column that is a dense sequence from 1 to maxRowID.
func makeRandomHandler(db *sql.DB, maxRowID int64) http.HandlerFunc {
	// Guard against invalid maxRowID to avoid rand.Int64N panic.
	if maxRowID <= 0 {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "no verses available", http.StatusNotFound)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		randomID := 1 + rand.Int64N(maxRowID)
		row := db.QueryRow("SELECT text FROM verses WHERE rowid >= ? LIMIT 1", randomID)
		var text string
		if err := row.Scan(&text); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			log.Printf("random query error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(text)
	}
}
