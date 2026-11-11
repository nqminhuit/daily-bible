package reading

import "testing"

func TestExistingSupportedCacheWithoutDayDetailsRemainsReadable(t *testing.T) {
	db := testDB(t)
	m := testManifest(testDay("2026-11-09", "dedication_of_the_lateran_basilica", "FEAST", "ordinary_time_32_monday", "YEAR_A", "YEAR_2"))
	importManifest(t, db, m)
	review(t, db, m.CalendarID, "2026-11-09", "Ga 2,13-22")
	if _, err := db.Exec(`DROP TABLE reading_calendar_day_details`); err != nil {
		t.Fatal(err)
	}
	got, err := LookupFor(t.Context(), db, "2026-11-09", m.CalendarID, DayMass)
	if err != nil || got.Ref != "Ga 2,13-22" || got.CalendarSourceURL != m.SourceURL {
		t.Fatal("old supported cache invalidated", got, err)
	}
}

func TestCorruptNationalDefinitionCannotServeOrBecomeCrawlCandidate(t *testing.T) {
	for _, payload := range []string{`broken`, `{"date":"2026-11-24"}`} {
		t.Run(payload, func(t *testing.T) {
			db := testDB(t)
			m := vietnamManifest()
			importManifest(t, db, m)
			review(t, db, m.CalendarID, martyrsDate, "Lc 9,23-26")
			if _, err := db.Exec(`UPDATE reading_calendar_day_details SET definition_json=? WHERE date='2026-11-24'`, payload); err != nil {
				t.Fatal(err)
			}
			if _, err := LookupFor(t.Context(), db, martyrsDate, m.CalendarID, DayMass); err == nil {
				t.Fatal("corrupt proof served")
			}
			if _, _, err := EntryForDate(t.Context(), db, m.CalendarID, martyrsDate); err == nil {
				t.Fatal("corrupt proof became crawl candidate")
			}
		})
	}
}
