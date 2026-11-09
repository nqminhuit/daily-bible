package main

import (
	"path/filepath"
	"testing"

	"github.com/minh/daily-bible/internal/constants"
)

func TestDatabasePath(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("DB_PATH", "")
		if got := databasePath(); got != constants.DBPath {
			t.Fatalf("databasePath() = %q, want %q", got, constants.DBPath)
		}
	})
	t.Run("override", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "custom-bible.db")
		t.Setenv("DB_PATH", path)
		if got := databasePath(); got != path {
			t.Fatalf("databasePath() = %q, want %q", got, path)
		}
	})
}
