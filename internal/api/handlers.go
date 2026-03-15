package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/minh/daily-bible/internal/model"
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
			http.Error(w, fmt.Sprintf("fts search error: %v", err), http.StatusInternalServerError)
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

func makeRandomHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		row := db.QueryRow(`SELECT text FROM verses ORDER BY RANDOM() LIMIT 1`)
		var text string
		if err := row.Scan(&text); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("random error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(text)
	}
}
