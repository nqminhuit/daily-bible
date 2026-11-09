package api

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func invalidReferences() []string {
	overflow := strconv.Itoa(math.MaxInt) + "0"
	return []string{
		"Ga 0,1", "Ga 1,0", "Ga 1,1-0", "Ga 1,3-2", "Ga 1,1c-1a",
		"Ga 1,1junk", "Ga 1,x1", "Ga 1,1-2junk", "Ga 1,1 2", "Ga 1,1,2",
		"Ga 1,1-", "Ga 1,-1", "Ga 1,.1", "Ga 1,1.", "Ga 1,1..2",
		"Ga 1,1-1,2", "Ga 2,1-1,2", "Ga 1,0-2,1", "Ga 1,1-2,0",
		"Ga " + overflow + ",1", "Ga 1," + overflow, "Ga 1,1-" + overflow,
		"Ga 1," + overflow + "a-2,1", "Ga 1,1-" + overflow + ",1", "Ga 1,1-2," + overflow,
		"Ga 1," + strings.Repeat("1.", maxRefSegments) + "1",
		fmt.Sprintf("Ga 1,1-%d,1", maxRefSegments+1),
		fmt.Sprintf("Ga 1,1.2-%d,1", maxRefSegments),
		"Ga 1,1" + strings.Repeat(" ", maxRefLength),
	}
}

func TestReferenceValidationBeforeDatabaseAccess(t *testing.T) {
	for _, ref := range invalidReferences() {
		t.Run(ref, func(t *testing.T) {
			if _, err := ParseRefMultiChapter(ref); err == nil {
				t.Fatal("expected parse error")
			}
			if _, _, _, err := ParseRef(ref); err == nil {
				t.Fatal("legacy parser accepted invalid reference")
			}
			// A nil database proves invalid input is rejected before query execution.
			if _, err := QueryByRef(t.Context(), nil, ref); err == nil {
				t.Fatal("expected query validation error")
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.URL.Path = "/api/v1/gospel/" + ref
			w := httptest.NewRecorder()
			makeGetGospelHandler(nil).ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestReferenceLimits(t *testing.T) {
	for _, ref := range []string{
		"Ga 1,1" + strings.Repeat(" ", maxRefLength-len("Ga 1,1")),
		"Ga 1," + strings.TrimSuffix(strings.Repeat("1.", maxRefSegments), "."),
		fmt.Sprintf("Ga 1,1-%d,1", maxRefSegments),
		fmt.Sprintf("Ga %d,1-%d,1", math.MaxInt-1, math.MaxInt),
		"  Ga 1,1 . 3a-4b  ",
	} {
		parsed, err := ParseRefMultiChapter(ref)
		if err != nil {
			t.Fatalf("valid boundary reference %q: %v", ref, err)
		}
		count := 0
		for _, segment := range parsed[0].Segments {
			count += len(segment.Ranges)
		}
		if count > maxRefSegments {
			t.Fatalf("expanded to %d ranges", count)
		}
	}
}

func TestQuerySuffixBoundaries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	for verse := 1; verse <= 3; verse++ {
		for _, suffix := range []string{"", "a", "b", "c"} {
			if _, err := db.Exec(`INSERT INTO verses(book, chapter, verse, verse_suffix, text) VALUES ('Ga', 1, ?, ?, 'text')`, verse, suffix); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, tc := range []struct {
		ref  string
		want []string
	}{
		{"Ga 1,1-3", []string{"1", "2", "3"}},
		{"Ga 1,1b-3b", []string{"1b", "1c", "2", "3", "3a", "3b"}},
		{"Ga 1,1-3b", []string{"1", "2", "3", "3a", "3b"}},
		{"Ga 1,1b-3", []string{"1b", "1c", "2", "3"}},
		{"Ga 1,1b-1c", []string{"1b", "1c"}},
		{"Ga 1,1b", []string{"1b", "1c"}},
		{"Ga 1,1-1b", []string{"1", "1a", "1b"}},
		{"Ga 1,1b-2a", []string{"1b", "1c", "2", "2a"}},
		{"Ga 1,1.3b", []string{"1", "3b", "3c"}},
		{"Ga 1,1-2.2-3", []string{"1", "2", "3"}},
	} {
		t.Run(tc.ref, func(t *testing.T) {
			verses, err := QueryByRef(t.Context(), db, tc.ref)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, v := range verses {
				got = append(got, strconv.Itoa(v.Verse)+v.VerseSuffix)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestQueryLargeSuffixRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	for _, row := range []struct {
		verse  int
		suffix string
	}{{1, "a"}, {1000, ""}, {math.MaxInt, "b"}, {math.MaxInt, "c"}} {
		if _, err := db.Exec(`INSERT INTO verses(book, chapter, verse, verse_suffix, text) VALUES ('Ga', 1, ?, ?, 'text')`, row.verse, row.suffix); err != nil {
			t.Fatal(err)
		}
	}
	ref := fmt.Sprintf("Ga 1,1a-%db", math.MaxInt)
	verses, err := QueryByRef(t.Context(), db, ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(verses) != 3 || verses[2].Verse != math.MaxInt || verses[2].VerseSuffix != "b" {
		t.Fatalf("unexpected verses: %+v", verses)
	}
	// The maximum allowed segment count also remains below SQLite expression and
	// bind-variable limits, even when every range has both suffix boundaries.
	ref = "Ga 1," + strings.TrimSuffix(strings.Repeat(fmt.Sprintf("1a-%db.", math.MaxInt), maxRefSegments), ".")
	verses, err = QueryByRef(t.Context(), db, ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(verses) != 3 {
		t.Fatalf("overlapping ranges returned %d verses", len(verses))
	}
}

func TestCrossChapterSuffixRanges(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	for _, row := range []struct {
		chapter, verse int
		suffix         string
	}{
		{1, 2, "a"}, {1, 2, "b"}, {1, 1000, ""}, {2, 1, ""}, {2, 1, "a"}, {3, 1, ""}, {3, 1, "a"}, {3, 1, "b"}, {3, 3, ""},
	} {
		if _, err := db.Exec(`INSERT INTO verses(book, chapter, verse, verse_suffix, text) VALUES ('Ga', ?, ?, ?, 'text')`, row.chapter, row.verse, row.suffix); err != nil {
			t.Fatal(err)
		}
	}
	verses, err := QueryByRef(t.Context(), db, "Ga 1,2b-3,1a.3")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range verses {
		got = append(got, fmt.Sprintf("%d,%d%s", v.Chapter, v.Verse, v.VerseSuffix))
	}
	want := []string{"1,2b", "1,1000", "2,1", "3,1", "3,1a", "3,3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func FuzzParseRefMultiChapter(f *testing.F) {
	for _, ref := range append(invalidReferences(), "Ga 1,1", "Mt 18,21-19,1", "Ga 1,1a-3b.5") {
		f.Add(ref)
	}
	f.Fuzz(func(t *testing.T, ref string) {
		parsed, err := ParseRefMultiChapter(ref)
		if err != nil {
			return
		}
		if len(parsed) != 1 || len(parsed[0].Segments) == 0 || len(ref) > maxRefLength {
			t.Fatalf("invalid result: %+v", parsed)
		}
		count := 0
		for _, segment := range parsed[0].Segments {
			if segment.Chapter <= 0 || len(segment.Ranges) == 0 {
				t.Fatalf("invalid segment: %+v", segment)
			}
			for _, r := range segment.Ranges {
				count++
				if r.Start <= 0 || r.End < r.Start {
					t.Fatalf("invalid range: %+v", r)
				}
			}
		}
		if count > maxRefSegments {
			t.Fatalf("too many ranges: %d", count)
		}
	})
}
