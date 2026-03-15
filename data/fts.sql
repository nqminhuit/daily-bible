-- Full text search using FTS5
CREATE VIRTUAL TABLE IF NOT EXISTS verses_fts
USING fts5(
  text,
  content='verses',
  content_rowid='rowid',
  tokenize='unicode61 remove_diacritics 2',
  prefix='2 3 4',
);
