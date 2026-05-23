-- Migration 002: daily_readings table
-- Replaces 002_lectionary.sql and 003_calendar.sql
-- Stores date -> lectionary_key -> gospel_ref in a single table
-- gospel_ref is NULL until the lectionary crawler populates it
CREATE TABLE IF NOT EXISTS daily_readings (
    date           TEXT NOT NULL PRIMARY KEY,
    lectionary_key TEXT NOT NULL,
    gospel_ref     TEXT
);
