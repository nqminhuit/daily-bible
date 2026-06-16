package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/api"
	"github.com/minh/daily-bible/internal/constants"
	"github.com/minh/daily-bible/internal/dateutil"
	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/query"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		usage()
		return
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = constants.DBPath
	}

	db, err := dbpkg.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch cmd {
	case "today":
		handleToday(ctx, db)
	case "gospel":
		if len(os.Args) < 3 {
			log.Fatal("usage: daily-bible gospel <ref>")
		}
		handleGospel(ctx, db, os.Args[2])
	case "search":
		if len(os.Args) < 3 {
			log.Fatal("usage: daily-bible search <query>")
		}
		handleSearch(ctx, db, os.Args[2])
	case "random":
		handleRandom(ctx, db)
	default:
		handleDate(ctx, db, cmd)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: daily-bible <command> [args]

Commands:
  today              Today's Gospel reading
  gospel <ref>       Gospel verses by reference (e.g. "Ga 9,1-41")
  search <query>     Full-text phrase search
  random             Random verse
  <date>             Gospel reading for a date (e.g. 2026-03-22)
  help               Show this help
`)
}

func handleToday(ctx context.Context, db *sql.DB) {
	outputDateReading(ctx, db, dateutil.TodayDate())
}

func handleDate(ctx context.Context, db *sql.DB, date string) {
	outputDateReading(ctx, db, date)
}

func outputDateReading(ctx context.Context, db *sql.DB, date string) {
	var lectionaryKey, gospelRef string
	err := db.QueryRowContext(ctx, `
		SELECT lectionary_key, gospel_ref
		FROM daily_readings
		WHERE date = ? AND gospel_ref IS NOT NULL
	`, date).Scan(&lectionaryKey, &gospelRef)

	if err == sql.ErrNoRows {
		log.Fatalf("no reading found for date %s", date)
	}
	if err != nil {
		log.Fatalf("query error: %v", err)
	}

	verses, err := api.QueryByRef(ctx, db, gospelRef)
	if err != nil {
		log.Fatalf("verses query error for ref %q: %v", gospelRef, err)
	}
	if len(verses) == 0 {
		log.Fatal("not found")
	}

	var sb strings.Builder
	for _, v := range verses {
		sb.WriteString(v.Text)
		sb.WriteByte(' ')
	}
	combined := strings.TrimSpace(sb.String())

	json.NewEncoder(os.Stdout).Encode(map[string]any{
		"date":           date,
		"lectionary_key": lectionaryKey,
		"ref":            gospelRef,
		"verses":         combined,
	})
}

func handleGospel(ctx context.Context, db *sql.DB, ref string) {
	verses, err := api.QueryByRef(ctx, db, ref)
	if err != nil {
		log.Fatalf("query error: %v", err)
	}
	if len(verses) == 0 {
		log.Fatal("not found")
	}
	json.NewEncoder(os.Stdout).Encode(verses)
}

func handleSearch(ctx context.Context, db *sql.DB, q string) {
	if strings.TrimSpace(q) == "" {
		log.Fatal("empty query")
	}
	if len(q) > 200 {
		log.Fatal("query too long")
	}

	rows, err := db.QueryContext(ctx, `SELECT text FROM verses_fts WHERE verses_fts MATCH ? LIMIT 10`,
		query.FtsPhraseQuery(q))
	if err != nil {
		log.Fatalf("search error: %v", err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			log.Printf("scan error: %v", err)
			continue
		}
		results = append(results, text)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("rows error: %v", err)
	}
	json.NewEncoder(os.Stdout).Encode(results)
}

func handleRandom(ctx context.Context, db *sql.DB) {
	var maxRowID int64
	if err := db.QueryRowContext(ctx, "SELECT IFNULL(MAX(rowid), 0) FROM verses").Scan(&maxRowID); err != nil {
		log.Fatalf("query max rowid: %v", err)
	}
	if maxRowID <= 0 {
		log.Fatal("no verses available")
	}

	var text string
	for range 10 {
		randomID := 1 + rand.Int64N(maxRowID)
		row := db.QueryRowContext(ctx, "SELECT text FROM verses WHERE rowid = ?", randomID)
		if err := row.Scan(&text); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			log.Fatalf("random query error: %v", err)
		}
		json.NewEncoder(os.Stdout).Encode(text)
		return
	}
	log.Fatal("not found")
}
