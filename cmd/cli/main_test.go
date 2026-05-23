package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var cliPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "daily-bible-cli-test")
	if err != nil {
		os.Stderr.WriteString("setup: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	cliPath = filepath.Join(tmpDir, "daily-bible")
	cmd := exec.Command("go", "build", "-tags", "fts5", "-o", cliPath, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Stderr.WriteString("build failed: " + err.Error() + "\n" + string(out) + "\n")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func applyMigrations(t *testing.T, database *sql.DB) {
	t.Helper()
	migrationsDir := "../../data/migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 0 && e.Name()[0] >= '0' && e.Name()[0] <= '9' {
			content, err := os.ReadFile(filepath.Join(migrationsDir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := database.Exec(string(content)); err != nil {
				t.Fatalf("apply migration %s: %v", e.Name(), err)
			}
		}
	}
}

func setupTestDB(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "bible.db")

	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	database.Exec("PRAGMA journal_mode=WAL;")
	applyMigrations(t, database)

	_, err = database.Exec(`
		INSERT INTO verses (book, chapter, verse, verse_suffix, text) VALUES
		('Ga', 11, 1, '', 'Now a man was ill...'),
		('Ga', 11, 2, '', 'Mary was the one...'),
		('Ga', 11, 45, '', 'Many of the Jews...')
	`)
	if err != nil {
		t.Fatal(err)
	}

	today := time.Now().Format("2006-01-02")
	_, err = database.Exec(`
		INSERT INTO calendar (date, lectionary_key, season, sunday_cycle, weekday, weekday_cycle, week_of_season)
		VALUES (?, 'lent_5_sun_A', 'lent', 'A', 'sun', '', 5)
	`, today)
	if err != nil {
		t.Fatal(err)
	}

	_, err = database.Exec(`
		INSERT INTO lectionary (lectionary_key, gospel_ref)
		VALUES ('lent_5_sun_A', 'Ga 11,1-2')
	`)
	if err != nil {
		t.Fatal(err)
	}

	return dbPath
}

func runCLI(t *testing.T, dbPath string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(cliPath, args...)
	cmd.Env = append(os.Environ(), "DB_PATH="+dbPath)
	return cmd.CombinedOutput()
}

func TestCLI_Help(t *testing.T) {
	out, err := exec.Command(cliPath, "help").CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if len(out) == 0 {
		t.Fatal("expected help output")
	}
}

func TestCLI_NoArgs(t *testing.T) {
	err := exec.Command(cliPath).Run()
	if err == nil {
		t.Fatal("expected non-zero exit for no args")
	}
}

func TestCLI_Gospel(t *testing.T) {
	dbPath := setupTestDB(t)
	out, err := runCLI(t, dbPath, "gospel", "Ga 11,1-2")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var results []map[string]any
	if err := json.Unmarshal(out, &results); err != nil {
		t.Fatalf("json unmarshal: %v\nbody: %s", err, out)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 verses, got %d", len(results))
	}
	if results[0]["verse"] != float64(1) {
		t.Fatalf("expected verse 1, got %v", results[0]["verse"])
	}
}

func TestCLI_Gospel_NotFound(t *testing.T) {
	dbPath := setupTestDB(t)
	out, err := runCLI(t, dbPath, "gospel", "Ga 99,1-2")
	if err == nil {
		t.Fatal("expected error for not found gospel ref")
	}
	if string(out) == "" {
		t.Fatal("expected error message on stderr")
	}
}

func TestCLI_Date(t *testing.T) {
	dbPath := setupTestDB(t)
	today := time.Now().Format("2006-01-02")

	out, err := runCLI(t, dbPath, today)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("json unmarshal: %v\nbody: %s", err, out)
	}
	if result["date"] != today {
		t.Fatalf("expected date %q, got %q", today, result["date"])
	}
	if result["lectionary_key"] != "lent_5_sun_A" {
		t.Fatalf("unexpected lectionary_key: %v", result["lectionary_key"])
	}
}

func TestCLI_Today(t *testing.T) {
	dbPath := setupTestDB(t)
	today := time.Now().Format("2006-01-02")

	out, err := runCLI(t, dbPath, "today")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("json unmarshal: %v\nbody: %s", err, out)
	}
	if result["date"] != today {
		t.Fatalf("expected date %q, got %q", today, result["date"])
	}
	if result["ref"] != "Ga 11,1-2" {
		t.Fatalf("unexpected ref: %v", result["ref"])
	}
}

func TestCLI_Date_NoData(t *testing.T) {
	dbPath := setupTestDB(t)
	out, err := runCLI(t, dbPath, "2099-01-01")
	if err == nil {
		t.Fatal("expected error for date with no reading")
	}
	if string(out) == "" {
		t.Fatal("expected error message on stderr")
	}
}

func TestCLI_Random(t *testing.T) {
	dbPath := setupTestDB(t)

	out, err := runCLI(t, dbPath, "random")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var text string
	if err := json.Unmarshal(out, &text); err != nil {
		t.Fatalf("json unmarshal: %v\nbody: %s", err, out)
	}
	if text == "" {
		t.Fatal("expected non-empty random verse")
	}
}

func TestCLI_Random_NoVerses(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "bible.db")
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	database.Exec("PRAGMA journal_mode=WAL;")
	applyMigrations(t, database)

	out, err := runCLI(t, dbPath, "random")
	if err == nil {
		t.Fatal("expected error for empty DB")
	}
	if string(out) == "" {
		t.Fatal("expected error message on stderr")
	}
}

func TestCLI_Search(t *testing.T) {
	dbPath := setupTestDB(t)

	out, err := runCLI(t, dbPath, "search", "man")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var results []string
	if err := json.Unmarshal(out, &results); err != nil {
		t.Fatalf("json unmarshal: %v\nbody: %s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one search result")
	}
}

func TestCLI_Search_Empty(t *testing.T) {
	dbPath := setupTestDB(t)

	_, err := runCLI(t, dbPath, "search", "")
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestCLI_Search_TooLong(t *testing.T) {
	dbPath := setupTestDB(t)

	long := make([]byte, 201)
	for i := range long {
		long[i] = 'a'
	}

	out, err := runCLI(t, dbPath, "search", string(long))
	if err == nil {
		t.Fatal("expected error for long query")
	}
	if string(out) == "" {
		t.Fatal("expected error message on stderr")
	}
}
