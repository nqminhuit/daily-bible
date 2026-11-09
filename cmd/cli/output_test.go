package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/minh/daily-bible/internal/dateutil"
)

func TestCLI_JSONOutputFailure(t *testing.T) {
	dbPath := setupTestDB(t)
	path := filepath.Join(t.TempDir(), "read-only-output")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	// A read-only descriptor reliably rejects writes, without relying on /dev/full
	// or platform-specific broken-pipe signal behavior.
	out, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	for _, args := range [][]string{{"today"}, {dateutil.TodayDate()}, {"gospel", "Ga 11,1"}, {"search", "man"}, {"random"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := exec.Command(cliPath, args...)
			cmd.Env = append(os.Environ(), "DB_PATH="+dbPath)
			cmd.Stdout = out
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err == nil {
				t.Fatal("expected nonzero exit on stdout write failure")
			}
			if !strings.Contains(stderr.String(), "write JSON output:") {
				t.Fatalf("missing write failure diagnostic: %s", stderr.String())
			}
		})
	}
}
