-- A calendar key is an estimate, never proof of a date's Mass reading.
-- Preserve legacy references for audit, but only an exact-date source record
-- matching the current reference makes a daily reading verified.
CREATE TABLE IF NOT EXISTS daily_reading_sources (
    date TEXT PRIMARY KEY REFERENCES daily_readings(date) ON DELETE CASCADE,
    gospel_ref TEXT NOT NULL,
    source_url TEXT NOT NULL,
    source_date TEXT NOT NULL CHECK (source_date = date),
    verified_at TEXT NOT NULL,
    celebration TEXT
);
