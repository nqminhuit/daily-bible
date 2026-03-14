package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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

		row := db.QueryRow(`SELECT reference, book, chapter_start, verse_start, chapter_end, verse_end, text FROM gospels WHERE reference = ? LIMIT 1`, ref)
		var g model.Gospel
		err = row.Scan(&g.Reference, &g.Book, &g.ChapterStart, &g.VerseStart, &g.ChapterEnd, &g.VerseEnd, &g.Text)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(g)
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

		rows, err := db.Query(`SELECT reference FROM gospels_fts WHERE gospels_fts MATCH ? LIMIT 10`,
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
		row := db.QueryRow(`SELECT reference, text FROM gospels ORDER BY RANDOM() LIMIT 1`)
		var ref, text string
		if err := row.Scan(&ref, &text); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("random error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"reference": ref, "text": text})
	}
}
