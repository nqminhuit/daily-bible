package scripture

import (
	"database/sql"
	"strconv"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestNormalizeGospelBoundsAndEquivalence(t *testing.T) {
	for _, ref := range []string{"Ga 2,13–22", "Mt 5,1-12a", "Ga 1,50-2,2"} {
		if _, err := NormalizeGospelRef(ref); err != nil {
			t.Fatalf("%s: %v", ref, err)
		}
	}
	for _, ref := range []string{"Rm 8,1-10", "Ga 22,1", "Ga 2,26", "Lc 1,81", "Ga 2,1-" + strconv.Itoa(EndOfChapter), "Ga 2,13-12"} {
		if _, err := NormalizeGospelRef(ref); err == nil {
			t.Fatalf("invalid reference accepted: %s", ref)
		}
	}
	if !Equivalent(" Ga   2,13–22 ", "Ga 2,13-22") || !Equivalent("Ga 2,13-13", "Ga 2,13") || Equivalent("Ga 2,13-22", "Ga 2,13-21") {
		t.Fatal("reference equivalence changed the selection")
	}
}
func TestHasTextRejectsGapsEmptyTextAndMissingChapterTail(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE verses(book TEXT,chapter INTEGER,verse INTEGER,verse_suffix TEXT,text TEXT);
	INSERT INTO verses VALUES ('Ga',1,50,'','text'),('Ga',1,51,'','text'),('Ga',2,1,'','text'),('Ga',2,2,'','text')`); err != nil {
		t.Fatal(err)
	}
	check := func(want bool) {
		t.Helper()
		ok, err := HasText(t.Context(), db, "Ga 1,50-2,2")
		if err != nil || ok != want {
			t.Fatalf("complete=%v, want %v, err=%v", ok, want, err)
		}
	}
	check(true)
	if _, err := db.Exec(`DELETE FROM verses WHERE chapter=1 AND verse=51`); err != nil {
		t.Fatal(err)
	}
	check(false)
	if _, err := db.Exec(`INSERT INTO verses VALUES ('Ga',1,51,'','')`); err != nil {
		t.Fatal(err)
	}
	check(false)
	if _, err := db.Exec(`UPDATE verses SET text='text'; DELETE FROM verses WHERE chapter=2 AND verse=1`); err != nil {
		t.Fatal(err)
	}
	check(false)
}
func TestHasTextRequiresRequestedSuffixEndpoints(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE verses(book TEXT,chapter INTEGER,verse INTEGER,verse_suffix TEXT,text TEXT); INSERT INTO verses VALUES ('Ga',2,13,'a','text')`); err != nil {
		t.Fatal(err)
	}
	if ok, err := HasText(t.Context(), db, "Ga 2,13a-13b"); err != nil || ok {
		t.Fatal("missing suffix accepted", ok, err)
	}
	if _, err := db.Exec(`INSERT INTO verses VALUES ('Ga',2,13,'b','text')`); err != nil {
		t.Fatal(err)
	}
	if ok, err := HasText(t.Context(), db, "Ga 2,13a-13b"); err != nil || !ok {
		t.Fatal("complete suffixes rejected", ok, err)
	}
	if _, err := db.Exec(`DELETE FROM verses WHERE verse_suffix='b'; INSERT INTO verses VALUES ('Ga',2,13,'c','text')`); err != nil {
		t.Fatal(err)
	}
	if ok, err := HasText(t.Context(), db, "Ga 2,13a-13c"); err != nil || ok {
		t.Fatal("missing interior suffix accepted", ok, err)
	}
}
