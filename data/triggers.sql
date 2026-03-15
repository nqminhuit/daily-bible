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
