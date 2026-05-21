-- Migration 003: Calendar table
-- Stores date -> lectionary_key mapping from liturgical calendar JSON
-- Note: No FK constraint to allow importing calendar before lectionary is populated
CREATE TABLE IF NOT EXISTS calendar (
    date           TEXT NOT NULL PRIMARY KEY,
    lectionary_key TEXT NOT NULL,
    season         TEXT NOT NULL,
    sunday_cycle   TEXT NOT NULL,
    weekday        TEXT NOT NULL,
    weekday_cycle  TEXT NOT NULL,
    week_of_season INTEGER NOT NULL
);
