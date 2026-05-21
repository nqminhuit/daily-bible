package api

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDBWithLectionary(t *testing.T) *sql.DB {
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

func TestParseRef(t *testing.T) {
	tests := []struct {
		ref        string
		wantBook   string
		wantCh     int
		wantRanges []verseRange
		wantErr    bool
	}{
		{
			ref:        "Ga 11,1-45",
			wantBook:   "Ga",
			wantCh:     11,
			wantRanges: []verseRange{{start: 1, end: 45}},
		},
		{
			ref:      "Mt 5,20-22a.27-28",
			wantBook: "Mt",
			wantCh:   5,
			wantRanges: []verseRange{
				{start: 20, startSuffix: "", end: 22, endSuffix: "a"},
				{start: 27, end: 28},
			},
		},
		{
			ref:        "Lc 1,1",
			wantBook:   "Lc",
			wantCh:     1,
			wantRanges: []verseRange{{start: 1, end: 1}},
		},
		{
			ref:     "invalid",
			wantErr: true,
		},
		{
			ref:     "Mt 5,",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			book, chapter, ranges, err := parseRef(tc.ref)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for ref %q", tc.ref)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if book != tc.wantBook {
				t.Errorf("book: got %q, want %q", book, tc.wantBook)
			}
			if chapter != tc.wantCh {
				t.Errorf("chapter: got %d, want %d", chapter, tc.wantCh)
			}
			if len(ranges) != len(tc.wantRanges) {
				t.Fatalf("ranges count: got %d, want %d", len(ranges), len(tc.wantRanges))
			}
			for i, r := range ranges {
				w := tc.wantRanges[i]
				if r.start != w.start || r.end != w.end || r.startSuffix != w.startSuffix || r.endSuffix != w.endSuffix {
					t.Errorf("range[%d]: got %+v, want %+v", i, r, w)
				}
			}
		})
	}
}

func TestQueryByRef(t *testing.T) {
	db := setupTestDBWithLectionary(t)

	_, err := db.Exec(`
		INSERT INTO verses (book, chapter, verse, verse_suffix, text) VALUES
		('Ga', 11, 1, '', 'Now a man was ill...'),
		('Ga', 11, 2, '', 'Mary was the one...'),
		('Ga', 11, 3, '', 'The sisters sent him...'),
		('Ga', 11, 45, '', 'Many of the Jews...'),
		('Mt', 5, 20, '', 'I tell you...'),
		('Mt', 5, 21, '', 'You have heard...'),
		('Mt', 5, 22, 'a', 'But I say to you...'),
		('Mt', 5, 27, '', 'You have heard...'),
		('Mt', 5, 28, '', 'But I say to you...')
	`)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("simple range", func(t *testing.T) {
		verses, err := queryByRef(t.Context(), db, "Ga 11,1-3")
		if err != nil {
			t.Fatalf("queryByRef error: %v", err)
		}
		if len(verses) != 3 {
			t.Fatalf("expected 3 verses, got %d", len(verses))
		}
		if verses[0].Verse != 1 || verses[1].Verse != 2 || verses[2].Verse != 3 {
			t.Fatalf("unexpected verses: %d, %d, %d", verses[0].Verse, verses[1].Verse, verses[2].Verse)
		}
	})

	t.Run("non-contiguous ranges", func(t *testing.T) {
		verses, err := queryByRef(t.Context(), db, "Mt 5,20-22a.27-28")
		if err != nil {
			t.Fatalf("queryByRef error: %v", err)
		}
		if len(verses) != 5 {
			t.Fatalf("expected 5 verses, got %d: %+v", len(verses), verses)
		}
		expected := []struct {
			verse  int
			suffix string
		}{
			{20, ""}, {21, ""}, {22, "a"}, {27, ""}, {28, ""},
		}
		for i, v := range verses {
			if v.Verse != expected[i].verse || v.VerseSuffix != expected[i].suffix {
				t.Errorf("verse[%d]: got %d%s, want %d%s", i, v.Verse, v.VerseSuffix, expected[i].verse, expected[i].suffix)
			}
		}
	})

	t.Run("single verse", func(t *testing.T) {
		verses, err := queryByRef(t.Context(), db, "Ga 11,45-45")
		if err != nil {
			t.Fatalf("queryByRef error: %v", err)
		}
		if len(verses) != 1 {
			t.Fatalf("expected 1 verse, got %d", len(verses))
		}
		if verses[0].Verse != 45 {
			t.Fatalf("expected verse 45, got %d", verses[0].Verse)
		}
	})
}

func TestTodayHandler(t *testing.T) {
	db := setupTestDBWithLectionary(t)

	_, err := db.Exec(`
		INSERT INTO verses (book, chapter, verse, verse_suffix, text) VALUES
		('Ga', 11, 1, '', 'Now a man was ill...'),
		('Ga', 11, 2, '', 'Mary was the one...'),
		('Ga', 11, 3, '', 'The sisters sent him...')
	`)
	if err != nil {
		t.Fatal(err)
	}

	today := time.Now().Format("2006-01-02")

	_, err = db.Exec(`
		INSERT INTO calendar (date, lectionary_key, season, sunday_cycle, weekday, weekday_cycle, week_of_season)
		VALUES (?, 'lent_5_sun_A', 'lent', 'A', 'sun', '', 5)
	`, today)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		INSERT INTO lectionary (lectionary_key, gospel_ref)
		VALUES ('lent_5_sun_A', 'Ga 11,1-3')
	`)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("today with data", func(t *testing.T) {
		handler := makeTodayHandler(db)
		req := httptest.NewRequest("GET", "/api/v1/today", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("date endpoint", func(t *testing.T) {
		handler := makeDateByPathHandler(db)
		req := httptest.NewRequest("GET", "/api/v1/date/"+today, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no reading for date", func(t *testing.T) {
		handler := makeDateByPathHandler(db)
		req := httptest.NewRequest("GET", "/api/v1/date/2099-01-01", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}
