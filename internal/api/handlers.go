package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/model"
)

var verseSegmentRe = regexp.MustCompile(`^(\d+)([a-zA-Z]?)$`)

func parseVerseSegments(seg string) (startNum int, startLetter string, endNum int, endLetter string, ok bool) {
	parts := strings.SplitN(seg, "-", 2)
	if len(parts) != 2 {
		return 0, "", 0, "", false
	}
	m1 := verseSegmentRe.FindStringSubmatch(strings.TrimSpace(parts[0]))
	m2 := verseSegmentRe.FindStringSubmatch(strings.TrimSpace(parts[1]))
	if m1 == nil || m2 == nil {
		return 0, "", 0, "", false
	}
	startNum, _ = strconv.Atoi(m1[1])
	startLetter = m1[2]
	endNum, _ = strconv.Atoi(m2[1])
	endLetter = m2[2]
	return startNum, startLetter, endNum, endLetter, true
}

func buildVerseCondition(segments []string) (string, []any, error) {
	if len(segments) == 0 {
		return "", nil, fmt.Errorf("no verse segments")
	}

	var clauses []string
	var args []any

	for _, seg := range segments {
		if strings.Contains(seg, "-") {
			sNum, sLet, eNum, eLet, ok := parseVerseSegments(seg)
			if !ok {
				return "", nil, fmt.Errorf("invalid verse segment: %s", seg)
			}
			if sLet == "" && eLet == "" {
				clauses = append(clauses, "verse BETWEEN ? AND ?")
				args = append(args, sNum, eNum)
			} else if sNum == eNum {
				clauses = append(clauses, "(verse = ? AND verse_suffix >= ? AND verse_suffix <= ?)")
				args = append(args, sNum, sLet, eLet)
			} else {
				var innerClauses []string
				var innerArgs []any
				for v := sNum; v <= eNum; v++ {
					if v == sNum && sLet != "" {
						innerClauses = append(innerClauses, "(verse = ? AND verse_suffix >= ?)")
						innerArgs = append(innerArgs, v, sLet)
					} else if v == eNum && eLet != "" {
						innerClauses = append(innerClauses, "(verse = ? AND verse_suffix <= ?)")
						innerArgs = append(innerArgs, v, eLet)
					} else {
						innerClauses = append(innerClauses, "verse = ?")
						innerArgs = append(innerArgs, v)
					}
				}
				clauses = append(clauses, "("+strings.Join(innerClauses, " OR ")+")")
				args = append(args, innerArgs...)
			}
		} else {
			m := verseSegmentRe.FindStringSubmatch(strings.TrimSpace(seg))
			if m == nil {
				return "", nil, fmt.Errorf("invalid verse: %s", seg)
			}
			vNum, _ := strconv.Atoi(m[1])
			vLet := m[2]
			if vLet != "" {
				clauses = append(clauses, "verse = ? AND verse_suffix = ?")
				args = append(args, vNum, vLet)
			} else {
				clauses = append(clauses, "verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL)")
				args = append(args, vNum)
			}
		}
	}

	return strings.Join(clauses, " OR "), args, nil
}

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

		parts := strings.Fields(ref)
		if len(parts) < 2 {
			http.Error(w, "invalid reference format", http.StatusBadRequest)
			return
		}

		bookIdx := 0
		for i, p := range parts {
			if strings.Contains(p, ",") {
				bookIdx = i
				break
			}
			if i == len(parts)-1 {
				http.Error(w, "invalid reference format", http.StatusBadRequest)
				return
			}
		}

		book := strings.Join(parts[:bookIdx], " ")
		chapterVerse := parts[bookIdx]

		chParts := strings.SplitN(chapterVerse, ",", 2)
		if len(chParts) != 2 {
			http.Error(w, "invalid reference format", http.StatusBadRequest)
			return
		}

		chapter, err := strconv.Atoi(chParts[0])
		if err != nil {
			http.Error(w, "invalid chapter", http.StatusBadRequest)
			return
		}

		versePart := chParts[1]
		segments := strings.Split(versePart, ".")

		verseCondition, args, err := buildVerseCondition(segments)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid verse reference: %v", err), http.StatusBadRequest)
			return
		}

		query := fmt.Sprintf(`
			SELECT book, chapter, verse, verse_suffix, text
			FROM verses
			WHERE book = ?
			AND chapter = ?
			AND (%s)
			ORDER BY verse, verse_suffix`, verseCondition)

		queryArgs := append([]any{book, chapter}, args...)

		rows, err := db.QueryContext(r.Context(), query, queryArgs...)
		if err != nil {
			http.Error(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var results []model.Gospel
		for rows.Next() {
			var g model.Gospel
			var suffix sql.NullString
			if err := rows.Scan(&g.Book, &g.Chapter, &g.Verse, &suffix, &g.Text); err != nil {
				http.Error(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
				return
			}
			if suffix.Valid {
				g.VerseSuffix = suffix.String
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
		if db == nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		rows, err := db.QueryContext(r.Context(), `SELECT text FROM verses_fts WHERE verses_fts MATCH ? LIMIT 10`,
			ftsPhraseQuery(q))
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

// ftsPhraseQuery wraps the query in double quotes for exact phrase matching in FTS5.
// This is intentional: the search endpoint is designed for phrase-only search.
// Users cannot search for individual tokens; the entire query is treated as a phrase.
func ftsPhraseQuery(q string) string {
	escaped := strings.ReplaceAll(q, `"`, `""`)
	return fmt.Sprintf(`"%s"`, escaped)
}

// makeRandomHandler returns a handler that serves a random verse from the database.
// The table "verses" is expected to be static, immutable,
// and have a rowid column that is a dense sequence from 1 to maxRowID.
func makeRandomHandler(db *sql.DB, maxRowID int64) http.HandlerFunc {
	if maxRowID <= 0 {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "no verses available", http.StatusNotFound)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var text string
		for range 10 {
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
		return time.Now().Format("2006-01-02")
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

		var lectionaryKey, gospelRef string
		err := db.QueryRowContext(r.Context(), `
			SELECT c.lectionary_key, l.gospel_ref
			FROM calendar c
			JOIN lectionary l ON c.lectionary_key = l.lectionary_key
			WHERE c.date = ?
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

		verses, err := queryByRef(r.Context(), db, gospelRef)
		if err != nil {
			log.Printf("verses query error for ref %q: %v", gospelRef, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"date":           date,
			"lectionary_key": lectionaryKey,
			"ref":            gospelRef,
			"verses":         verses,
		})
	}
}
