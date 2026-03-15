-- Fast lookup by reference
CREATE INDEX IF NOT EXISTS idx_verses_ref
ON verses(book, chapter, verse);

-- Optional: fast lookup by book
CREATE INDEX IF NOT EXISTS idx_verses_book
ON verses(book);
