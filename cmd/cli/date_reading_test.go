package main

import (
	"bytes"
	"database/sql"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/minh/daily-bible/internal/dateutil"
)

func TestCLI_DateReadingWithoutMatchingVerses(t *testing.T) {
	dbPath := setupTestDB(t)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`DELETE FROM verses`); err != nil {
		t.Fatal(err)
	}

	for _, arg := range []string{"today", dateutil.TodayDate()} {
		t.Run(arg, func(t *testing.T) {
			cmd := exec.Command(cliPath, arg)
			cmd.Env = append(os.Environ(), "DB_PATH="+dbPath)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err == nil {
				t.Fatal("expected nonzero exit for reading without matching verses")
			}
			if stdout.Len() != 0 {
				t.Fatalf("expected no success output, got %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), "not found") {
				t.Fatalf("expected not-found error, got %q", stderr.String())
			}
		})
	}
}
