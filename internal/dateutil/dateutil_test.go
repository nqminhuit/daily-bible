package dateutil

import (
	"testing"
	"time"

	"github.com/minh/daily-bible/internal/constants"
)

func TestTodayDate_Format(t *testing.T) {
	date := TodayDate()
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatalf("TodayDate() returned invalid date %q: %v", date, err)
	}
}

func TestTodayDate_MatchesToday(t *testing.T) {
	loc, err := time.LoadLocation(constants.Timezone)
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Now().In(loc).Format("2006-01-02")
	if got := TodayDate(); got != expected {
		t.Fatalf("TodayDate() = %q, want %q", got, expected)
	}
}

func TestToday_Timezone(t *testing.T) {
	now := today()
	loc, err := time.LoadLocation(constants.Timezone)
	if err != nil {
		t.Fatal(err)
	}
	if now.Location().String() != loc.String() {
		t.Fatalf("today().Location() = %q, want %q", now.Location(), loc.String())
	}
}

func TestTodayDate_ConsistentWithToday(t *testing.T) {
	date1 := TodayDate()
	date2 := today().Format("2006-01-02")
	if date1 != date2 {
		t.Fatalf("TodayDate() = %q, but today().Format() = %q", date1, date2)
	}
}
