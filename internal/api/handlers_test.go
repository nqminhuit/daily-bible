package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	migrationsDir := "../../data/migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 0 && e.Name()[0] >= '0' && e.Name()[0] <= '9' {
			content, err := os.ReadFile(migrationsDir + "/" + e.Name())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(string(content)); err != nil {
				t.Fatalf("apply migration %s: %v", e.Name(), err)
			}
		}
	}

	return db
}

func TestParseVerseSegments(t *testing.T) {
	tests := []struct {
		name               string
		seg                string
		wantStartNum       int
		wantStartLetter    string
		wantEndNum         int
		wantEndLetter      string
		wantOk             bool
	}{
		{
			name:            "simple range",
			seg:             "1-5",
			wantStartNum:    1,
			wantStartLetter: "",
			wantEndNum:      5,
			wantEndLetter:   "",
			wantOk:          true,
		},
		{
			name:            "range with start suffix",
			seg:             "1a-5",
			wantStartNum:    1,
			wantStartLetter: "a",
			wantEndNum:      5,
			wantEndLetter:   "",
			wantOk:          true,
		},
		{
			name:            "range with both suffixes",
			seg:             "1a-5b",
			wantStartNum:    1,
			wantStartLetter: "a",
			wantEndNum:      5,
			wantEndLetter:   "b",
			wantOk:          true,
		},
		{
			name:            "with spaces",
			seg:             " 1 -5 ",
			wantStartNum:    1,
			wantStartLetter: "",
			wantEndNum:      5,
			wantEndLetter:   "",
			wantOk:          true,
		},
		{
			name:   "no dash",
			seg:    "1",
			wantOk: false,
		},
		{
			name:   "empty string",
			seg:    "",
			wantOk: false,
		},
		{
			name:   "invalid start",
			seg:    "a-5",
			wantOk: false,
		},
		{
			name:   "invalid end",
			seg:    "1-b",
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sNum, sLet, eNum, eLet, ok := parseVerseSegments(tt.seg)
			if ok != tt.wantOk {
				t.Fatalf("parseVerseSegments(%q) ok = %v, want %v", tt.seg, ok, tt.wantOk)
			}
			if ok {
				if sNum != tt.wantStartNum || sLet != tt.wantStartLetter || eNum != tt.wantEndNum || eLet != tt.wantEndLetter {
					t.Fatalf("parseVerseSegments(%q) = (%d, %q, %d, %q), want (%d, %q, %d, %q)",
						tt.seg, sNum, sLet, eNum, eLet,
						tt.wantStartNum, tt.wantStartLetter, tt.wantEndNum, tt.wantEndLetter)
				}
			}
		})
	}
}

func TestBuildVerseCondition(t *testing.T) {
	tests := []struct {
		name          string
		segments      []string
		wantCondition string
		wantArgsLen   int
		wantErr       bool
	}{
		{
			name:          "single verse",
			segments:      []string{"1"},
			wantCondition: "verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL)",
			wantArgsLen:   1,
		},
		{
			name:          "single verse with suffix",
			segments:      []string{"1a"},
			wantCondition: "verse = ? AND verse_suffix = ?",
			wantArgsLen:   2,
		},
		{
			name:          "multiple verses",
			segments:      []string{"1", "2"},
			wantCondition: "verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL) OR verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL)",
			wantArgsLen:   2,
		},
		{
			name:          "simple range",
			segments:      []string{"1-5"},
			wantCondition: "(verse BETWEEN ? AND ? AND (verse_suffix = '' OR verse_suffix IS NULL))",
			wantArgsLen:   2,
		},
		{
			name:          "range with suffix same verse",
			segments:      []string{"1a-1b"},
			wantCondition: "(verse = ? AND verse_suffix >= ? AND verse_suffix <= ?)",
			wantArgsLen:   3,
		},
		{
			name:          "range cross verse with suffix",
			segments:      []string{"1a-3b"},
			wantCondition: "((verse = ? AND verse_suffix >= ?) OR (verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL)) OR (verse = ? AND verse_suffix <= ?))",
			wantArgsLen:   5,
		},
		{
			name:          "mixed single and range",
			segments:      []string{"1-3", "5"},
			wantCondition: "(verse BETWEEN ? AND ? AND (verse_suffix = '' OR verse_suffix IS NULL)) OR verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL)",
			wantArgsLen:   3,
		},
		{
			name:     "empty segments",
			segments: []string{},
			wantErr:  true,
		},
		{
			name:     "invalid segment",
			segments: []string{"abc"},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond, args, err := buildVerseCondition(tt.segments)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("buildVerseCondition(%v) expected error", tt.segments)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildVerseCondition(%v) unexpected error: %v", tt.segments, err)
			}
			if cond != tt.wantCondition {
				t.Fatalf("buildVerseCondition(%v) condition = %q, want %q", tt.segments, cond, tt.wantCondition)
			}
			if len(args) != tt.wantArgsLen {
				t.Fatalf("buildVerseCondition(%v) got %d args, want %d", tt.segments, len(args), tt.wantArgsLen)
			}
		})
	}
}

func TestGetGospel(t *testing.T) {
	db := setupTestDB(t)

	if _, err := db.Exec(`
	INSERT INTO verses(book,chapter,verse,verse_suffix,text)
	VALUES('Ga',10,31,'','Jews picked up stones...')`); err != nil {
		t.Fatal(err)
	}

	handler := makeGetGospelHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/gospel/Ga%2010,31-31", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}
