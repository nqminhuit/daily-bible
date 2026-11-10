-- Additive provenance and Mass/season separation. Keep the compact day table
-- compatible with existing readers; definitions never approve Gospel mappings.
CREATE TABLE IF NOT EXISTS reading_calendar_day_details (
    calendar_id TEXT NOT NULL,
    date TEXT NOT NULL,
    mass_option TEXT NOT NULL,
    rules_version TEXT NOT NULL,
    entry_key TEXT NOT NULL,
    definition_json TEXT NOT NULL,
    PRIMARY KEY (calendar_id, date, mass_option),
    FOREIGN KEY (calendar_id,date,mass_option)
        REFERENCES reading_calendar_days(calendar_id,date,mass_option)
);
