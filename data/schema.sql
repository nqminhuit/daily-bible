-- Each row = one verse
CREATE TABLE IF NOT EXISTS verses (
    book TEXT NOT NULL,
    chapter INTEGER NOT NULL,
    verse INTEGER NOT NULL,
    verse_suffix TEXT NOT NULL DEFAULT '',
    text TEXT NOT NULL,
    PRIMARY KEY(book, chapter, verse, verse_suffix)
);
