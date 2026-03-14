package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/minh/daily-bible/internal/constants"
	internaldb "github.com/minh/daily-bible/internal/db"
)

func main() {
	resp, err := http.Get(constants.GospelURL)
	if err != nil {
		log.Fatalf("download json: %v\n", err)
	}
	defer resp.Body.Close()

	if c := resp.StatusCode; c != http.StatusOK {
		log.Fatalf("bad http status: %d\n", c)
	}

	var m map[string]string
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("read response body: %v\n", err)
	}
	if err := json.Unmarshal(b, &m); err != nil {
		log.Fatalf("parse json: %v\n", err)
	}

	refs := make([]string, 0, len(m))
	for k := range m {
		refs = append(refs, k)
	}
	sort.Strings(refs)

	// write SQL file
	if err := os.MkdirAll(filepath.Dir(constants.DBImportPath), 0755); err != nil {
		log.Fatalf("mkdir: %v\n", err)
	}

	f, err := os.Create(constants.DBImportPath)
	if err != nil {
		log.Fatalf("create out: %v\n", err)
	}
	defer f.Close()

	fmt.Fprintln(f, "BEGIN TRANSACTION;")
	for _, ref := range refs {
		text := m[ref]
		book, cs, vs, ce, ve := parseReference(ref)
		escRef := escapeSQL(ref)
		escBook := escapeSQL(book)
		escText := escapeSQL(text)
		fmt.Fprintf(f, "INSERT INTO gospels(reference, book, chapter_start, verse_start, chapter_end, verse_end, text) VALUES ('%s','%s',%d,%d,%d,%d,'%s');\n",
			escRef, escBook, cs, vs, ce, ve, escText)
	}
	fmt.Fprintln(f, "COMMIT;")
	fmt.Printf("wrote %s\n", constants.DBImportPath)

	if err := os.MkdirAll(filepath.Dir(constants.DBPath), 0755); err != nil {
		log.Fatalf("mkdir db dir: %v\n", err)
	}
	dbConn, err := internaldb.Open(constants.DBPath)
	if err != nil {
		log.Fatalf("open db: %v\n", err)
	}
	defer dbConn.Close()

	if err := internaldb.InitDB(dbConn, constants.DBSchemaPath); err != nil {
		log.Fatalf("Error: init schema: %v", err)
	}

	if err := loadData(dbConn, refs, m); err != nil {
		log.Fatalf("load data: %v\n", err)
	}
	fmt.Printf("loaded %d rows into %s\n", len(refs), constants.DBPath)
}

func loadData(db *sql.DB, refs []string, m map[string]string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO gospels(reference, book, chapter_start, verse_start, chapter_end, verse_end, text) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, ref := range refs {
		text := m[ref]
		book, cs, vs, ce, ve := parseReference(ref)
		if _, err := stmt.Exec(ref, book, cs, vs, ce, ve, text); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func escapeSQL(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	return s
}

func parseReference(ref string) (book string, chapterStart, verseStart, chapterEnd, verseEnd int) {
	chapterStart, verseStart, chapterEnd, verseEnd = 0, 0, 0, 0
	ref = strings.TrimSpace(ref)
	parts := strings.SplitN(ref, " ", 2)
	book = parts[0]
	if len(parts) < 2 {
		return
	}
	body := strings.TrimSpace(parts[1])
	if strings.Contains(body, "-") {
		pieces := strings.SplitN(body, "-", 2)
		left := strings.TrimSpace(pieces[0])
		right := strings.TrimSpace(pieces[1])
		if strings.Contains(left, ",") {
			lp := strings.SplitN(left, ",", 2)
			chapterStart = atoiOrZero(lp[0])
			verseStart = atoiOrZero(lp[1])
		} else {
			chapterStart = atoiOrZero(left)
			verseStart = 0
		}
		if strings.Contains(right, ",") {
			rp := strings.SplitN(right, ",", 2)
			chapterEnd = atoiOrZero(rp[0])
			verseEnd = atoiOrZero(rp[1])
		} else {
			chapterEnd = chapterStart
			verseEnd = atoiOrZero(right)
		}
	} else {
		if strings.Contains(body, ",") {
			p := strings.SplitN(body, ",", 2)
			chapterStart = atoiOrZero(p[0])
			verseStart = atoiOrZero(p[1])
			chapterEnd = chapterStart
			verseEnd = verseStart
		} else {
			chapterStart = atoiOrZero(body)
			verseStart = 0
			chapterEnd = chapterStart
			verseEnd = verseStart
		}
	}
	if chapterEnd == 0 {
		chapterEnd = chapterStart
	}
	if verseEnd == 0 {
		verseEnd = verseStart
	}
	return
}

func atoiOrZero(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}
