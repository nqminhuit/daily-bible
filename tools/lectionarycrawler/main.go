package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/minh/daily-bible/internal/constants"
	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/parser"
)

func main() {
	db, err := dbpkg.Open(constants.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT DISTINCT c.lectionary_key, c.date
		FROM calendar c
		LEFT JOIN lectionary l ON c.lectionary_key = l.lectionary_key
		WHERE l.lectionary_key IS NULL
		GROUP BY c.lectionary_key
		ORDER BY c.date
	`)
	if err != nil {
		log.Fatalf("query missing keys: %v", err)
	}
	defer rows.Close()

	type job struct {
		key  string
		date string
	}
	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.key, &j.date); err != nil {
			log.Printf("scan: %v", err)
			continue
		}
		jobs = append(jobs, j)
	}
	rows.Close()

	log.Printf("found %d lectionary keys to populate", len(jobs))

	client := &http.Client{Timeout: 10 * time.Second}

	for i, j := range jobs {
		parts := strings.SplitN(j.date, "-", 3)
		if len(parts) != 3 {
			log.Printf("invalid date %q", j.date)
			continue
		}
		url := fmt.Sprintf(
			"https://www.vaticannews.va/vi/loi-chua-hang-ngay/%s/%s/%s.html",
			parts[0], parts[1], parts[2],
		)

		ref, err := fetchGospelRef(client, url)
		if err != nil {
			log.Printf("[%d/%d] FAIL key=%s date=%s: %v", i+1, len(jobs), j.key, j.date, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if _, err := db.Exec(
			`INSERT OR IGNORE INTO lectionary (lectionary_key, gospel_ref) VALUES (?, ?)`,
			j.key, ref,
		); err != nil {
			log.Printf("insert %s: %v", j.key, err)
			continue
		}

		log.Printf("[%d/%d] OK key=%s ref=%s", i+1, len(jobs), j.key, ref)
		time.Sleep(300 * time.Millisecond)
	}
}

func fetchGospelRef(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	_, ref, err := parser.ExtractGospel(string(body))
	return ref, err
}
