CREATE TABLE IF NOT EXISTS lectionary (
    date TEXT NOT NULL PRIMARY KEY,
    book TEXT NOT NULL,
    chapter INTEGER NOT NULL,
    verse_start INTEGER NOT NULL,
    verse_start_suffix TEXT NOT NULL DEFAULT '',
    verse_end INTEGER NOT NULL,
    verse_end_suffix TEXT NOT NULL DEFAULT ''
);
