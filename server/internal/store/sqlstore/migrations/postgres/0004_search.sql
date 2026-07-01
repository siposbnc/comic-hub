-- 0004_search (Postgres dialect): full-text search over series names and book titles.
-- Where SQLite uses FTS5 shadow tables + triggers, Postgres indexes the base tables
-- directly with GIN expression indexes — no shadow state to keep in lockstep. The
-- search repository queries with exactly these expressions so the indexes are used.
-- The 'simple' configuration matches FTS5's unicode61 tokenizer (no language stemming);
-- diacritic folding (remove_diacritics) is not replicated — accented queries must match
-- accented titles on Postgres.

CREATE INDEX idx_series_fts ON series USING GIN (to_tsvector('simple', name));
CREATE INDEX idx_book_fts ON book USING GIN (to_tsvector('simple', coalesce(title, '')));
