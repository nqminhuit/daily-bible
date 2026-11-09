-- Source observations and reviewed reusable mappings are different evidence.
CREATE TABLE IF NOT EXISTS reading_calendars (
    calendar_id TEXT PRIMARY KEY,
    rules_version TEXT NOT NULL,
    source_url TEXT NOT NULL,
    imported_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS reading_calendar_days (
    calendar_id TEXT NOT NULL REFERENCES reading_calendars(calendar_id),
    date TEXT NOT NULL,
    mass_option TEXT NOT NULL,
    rules_version TEXT NOT NULL,
    entry_key TEXT NOT NULL,
    observance TEXT NOT NULL,
    celebration TEXT NOT NULL DEFAULT '',
    displaced_entry TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (calendar_id, date, mass_option)
);
CREATE INDEX IF NOT EXISTS reading_calendar_days_entry ON reading_calendar_days(calendar_id, rules_version, entry_key, mass_option);
CREATE TABLE IF NOT EXISTS reading_entry_reviews (
    id INTEGER PRIMARY KEY,
    calendar_id TEXT NOT NULL,
    rules_version TEXT NOT NULL,
    entry_key TEXT NOT NULL,
    mass_option TEXT NOT NULL,
    gospel_ref TEXT NOT NULL,
    evidence_url TEXT NOT NULL,
    evidence_date TEXT,
    review_note TEXT NOT NULL,
    reviewed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS reading_entries (
    calendar_id TEXT NOT NULL,
    rules_version TEXT NOT NULL,
    entry_key TEXT NOT NULL,
    mass_option TEXT NOT NULL,
    review_id INTEGER NOT NULL REFERENCES reading_entry_reviews(id),
    PRIMARY KEY (calendar_id, rules_version, entry_key, mass_option)
);
CREATE TABLE IF NOT EXISTS reading_date_reviews (
    id INTEGER PRIMARY KEY,
    calendar_id TEXT NOT NULL,
    rules_version TEXT NOT NULL,
    date TEXT NOT NULL,
    mass_option TEXT NOT NULL,
    gospel_ref TEXT NOT NULL,
    evidence_url TEXT NOT NULL,
    evidence_date TEXT NOT NULL CHECK(evidence_date=date),
    review_note TEXT NOT NULL,
    celebration TEXT NOT NULL DEFAULT '',
    reviewed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS reading_date_acceptances (
    calendar_id TEXT NOT NULL,
    date TEXT NOT NULL,
    mass_option TEXT NOT NULL,
    review_id INTEGER NOT NULL REFERENCES reading_date_reviews(id),
    PRIMARY KEY(calendar_id,date,mass_option)
);
CREATE TABLE IF NOT EXISTS reading_observations (
    id INTEGER PRIMARY KEY,
    date TEXT NOT NULL,
    calendar_id TEXT NOT NULL,
    mass_option TEXT NOT NULL,
    gospel_ref TEXT NOT NULL,
    source_url TEXT NOT NULL,
    source_date TEXT NOT NULL CHECK(source_date = date),
    observed_at TEXT NOT NULL,
    UNIQUE(date, calendar_id, mass_option, gospel_ref, source_url, observed_at)
);
-- Preserve existing evidence without promoting it to a general reading rule.
INSERT OR IGNORE INTO reading_observations
    (date,calendar_id,mass_option,gospel_ref,source_url,source_date,observed_at)
SELECT date,'vatican-vi','day',gospel_ref,source_url,source_date,verified_at
FROM daily_reading_sources s
WHERE NOT EXISTS (
    SELECT 1 FROM reading_observations o
    WHERE o.date=s.date AND o.calendar_id='vatican-vi' AND o.mass_option='day'
      AND o.gospel_ref=s.gospel_ref AND o.source_url=s.source_url AND o.source_date=s.source_date
);
