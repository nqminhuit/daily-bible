-- Migration 001: Initial schema (verses + FTS + triggers)
-- Combined from schema.sql, fts.sql, and triggers.sql

-- Each row = one verse
CREATE TABLE IF NOT EXISTS verses (
    book TEXT NOT NULL,
    chapter INTEGER NOT NULL,
    verse INTEGER NOT NULL,
    verse_suffix TEXT NOT NULL DEFAULT '',
    text TEXT NOT NULL,
    PRIMARY KEY(book, chapter, verse, verse_suffix)
);

-- Full text search using FTS5
CREATE VIRTUAL TABLE IF NOT EXISTS verses_fts
USING fts5(
  text,
  content='verses',
  content_rowid='rowid',
  tokenize='unicode61 remove_diacritics 2',
  prefix='2 3 4',
);

-- Triggers to keep FTS index in sync
CREATE TRIGGER IF NOT EXISTS verses_ai
AFTER INSERT ON verses BEGIN
  INSERT INTO verses_fts(rowid, text)
  VALUES (new.rowid, new.text);
END;

CREATE TRIGGER IF NOT EXISTS verses_ad
AFTER DELETE ON verses BEGIN
  INSERT INTO verses_fts(verses_fts, rowid, text)
  VALUES('delete', old.rowid, old.text);
END;

CREATE TRIGGER IF NOT EXISTS verses_au
AFTER UPDATE ON verses BEGIN
  INSERT INTO verses_fts(verses_fts, rowid, text)
  VALUES('delete', old.rowid, old.text);
  INSERT INTO verses_fts(rowid, text)
  VALUES(new.rowid, new.text);
END;
