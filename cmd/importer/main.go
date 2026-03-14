package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	internaldb "github.com/minh/daily-bible/internal/db"
)

func main() {
	in := flag.String("in", "data/gospel.json", "input JSON file")
	out := flag.String("out", "data/import_gospels.sql", "output SQL file")
	schema := flag.String("schema", "data/schema.sql", "schema SQL file")
	dbPath := flag.String("db", "", "path to sqlite db to create and load data into (optional)")
	flag.Parse()

	b, err := ioutil.ReadFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}

	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		fmt.Fprintf(os.Stderr, "parse json: %v\n", err)
		os.Exit(1)
	}

	refs := make([]string, 0, len(m))
	for k := range m {
		refs = append(refs, k)
	}
	sort.Strings(refs)

	// write SQL file
	if err := os.MkdirAll(filepath.Dir(*out), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create out: %v\n", err)
		os.Exit(1)
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
	fmt.Printf("wrote %s\n", *out)

	// optionally load into sqlite db
	if *dbPath != "" {
		if err := os.MkdirAll(filepath.Dir(*dbPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "mkdir db dir: %v\n", err)
			os.Exit(1)
		}
		dbConn, err := internaldb.Open(*dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open db: %v\n", err)
			os.Exit(1)
		}
		defer dbConn.Close()

		if err := internaldb.InitDB(dbConn, *schema); err != nil {
			fmt.Fprintf(os.Stderr, "init db schema: %v\n", err)
			os.Exit(1)
		}

		if err := loadData(dbConn, refs, m); err != nil {
			fmt.Fprintf(os.Stderr, "load data: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("loaded %d rows into %s\n", len(refs), *dbPath)
	}
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
