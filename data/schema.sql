-- Each row = one verse
CREATE TABLE IF NOT EXISTS verses (
    id INTEGER PRIMARY KEY,
    book TEXT NOT NULL,
    chapter INTEGER NOT NULL,
    verse INTEGER NOT NULL,
    text TEXT NOT NULL,
    UNIQUE(book, chapter, verse)
);
