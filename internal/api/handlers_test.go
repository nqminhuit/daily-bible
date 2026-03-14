package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/minh/daily-bible/internal/api"
	dbpkg "github.com/minh/daily-bible/internal/db"
	"github.com/minh/daily-bible/internal/model"
)

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", err
		}
		dir = parent
	}
}

func TestHandlers(t *testing.T) {
	tmpdb := filepath.Join(os.TempDir(), "daily_bible_api_test.db")
	defer os.Remove(tmpdb)
	os.Remove(tmpdb)

	dbConn, err := dbpkg.Open(tmpdb)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer dbConn.Close()

	// locate repo root to find data/schema.sql
	root, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	schemaPath := filepath.Join(root, "data", "schema.sql")
	if err := dbpkg.InitDB(dbConn, schemaPath); err != nil {
		t.Fatalf("init db: %v", err)
	}

	_, err = dbConn.Exec(`INSERT INTO gospels(reference, book, chapter_start, verse_start, chapter_end, verse_end, text) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"Mt 1,1-2", "Mt", 1, 1, 1, 2, "In the beginning Jesus")
	if err != nil {
		t.Fatalf("insert sample: %v", err)
	}

	router := api.NewRouter(dbConn)
	ts := httptest.NewServer(router)
	defer ts.Close()

	// GET gospel
	resp, err := http.Get(ts.URL + "/api/v1/gospel/" + url.PathEscape("Mt 1,1-2"))
	if err != nil {
		t.Fatalf("get gospel: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var g model.Gospel
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		t.Fatalf("decode gospel: %v", err)
	}
	resp.Body.Close()
	if g.Reference != "Mt 1,1-2" {
		t.Fatalf("unexpected ref %s", g.Reference)
	}

	// search
	resp, err = http.Get(ts.URL + "/api/v1/search?q=Jesus")
	if err != nil {
		t.Fatalf("search err: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var results []string
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	resp.Body.Close()
	if len(results) == 0 {
		t.Fatalf("expected results")
	}

	// random
	resp, err = http.Get(ts.URL + "/api/v1/random")
	if err != nil {
		t.Fatalf("random err: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var rnd map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&rnd); err != nil {
		t.Fatalf("decode random: %v", err)
	}
	resp.Body.Close()
	if rnd["reference"] == "" {
		t.Fatalf("empty reference in random")
	}
}
