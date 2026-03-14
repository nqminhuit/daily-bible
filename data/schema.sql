-- Schema for gospels
CREATE TABLE IF NOT EXISTS gospels (
    id INTEGER PRIMARY KEY,
    reference TEXT UNIQUE NOT NULL,
    book TEXT,
    chapter_start INTEGER,
    verse_start INTEGER,
    chapter_end INTEGER,
    verse_end INTEGER,
    text TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gospel_book
ON gospels(book);

-- Full text search using FTS5
CREATE VIRTUAL TABLE IF NOT EXISTS gospels_fts
USING fts5(
  reference,
  text,
  content='gospels',
  content_rowid='id'
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER IF NOT EXISTS gospels_ai AFTER INSERT ON gospels BEGIN
  INSERT INTO gospels_fts(rowid, reference, text)
  VALUES (new.id, new.reference, new.text);
END;

CREATE TRIGGER IF NOT EXISTS gospels_ad AFTER DELETE ON gospels BEGIN
  INSERT INTO gospels_fts(gospels_fts, rowid, reference, text)
  VALUES('delete', old.id, old.reference, old.text);
END;

CREATE TRIGGER IF NOT EXISTS gospels_au AFTER UPDATE ON gospels BEGIN
  INSERT INTO gospels_fts(gospels_fts, rowid, reference, text)
  VALUES('delete', old.id, old.reference, old.text);
  INSERT INTO gospels_fts(rowid, reference, text)
  VALUES(new.id, new.reference, new.text);
END;
