-- Migration 002: Lectionary table
-- Maps lectionary_key to gospel_ref (e.g. "ordinary_3_mon_II" -> "Ga 11,1-45")
CREATE TABLE IF NOT EXISTS lectionary (
    lectionary_key TEXT NOT NULL PRIMARY KEY,
    gospel_ref     TEXT NOT NULL
);
