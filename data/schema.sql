-- Each row = one verse
CREATE TABLE IF NOT EXISTS verses (
    book TEXT NOT NULL,
    chapter INTEGER NOT NULL,
    verse INTEGER NOT NULL,
    text TEXT NOT NULL,
    PRIMARY KEY(book, chapter, verse)
);
