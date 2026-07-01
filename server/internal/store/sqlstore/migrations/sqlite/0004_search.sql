-- 0004_search: full-text search over series names and book titles (docs/03-api.md §8).
-- Two standalone FTS5 tables carry the searchable text plus the owning row's TEXT id (as
-- an UNINDEXED column, so results map back to series/book). Triggers keep them in lockstep
-- with the base tables; the inserts at the end backfill rows that already exist.

CREATE VIRTUAL TABLE series_fts USING fts5(
  series_id UNINDEXED,
  name,
  tokenize = 'unicode61 remove_diacritics 2'
);

CREATE VIRTUAL TABLE book_fts USING fts5(
  book_id UNINDEXED,
  title,
  tokenize = 'unicode61 remove_diacritics 2'
);

-- ── series → series_fts ───────────────────────────────────────────────
CREATE TRIGGER series_fts_ai AFTER INSERT ON series BEGIN
  INSERT INTO series_fts (series_id, name) VALUES (new.id, new.name);
END;
CREATE TRIGGER series_fts_ad AFTER DELETE ON series BEGIN
  DELETE FROM series_fts WHERE series_id = old.id;
END;
CREATE TRIGGER series_fts_au AFTER UPDATE OF name ON series BEGIN
  DELETE FROM series_fts WHERE series_id = old.id;
  INSERT INTO series_fts (series_id, name) VALUES (new.id, new.name);
END;

-- ── book → book_fts ───────────────────────────────────────────────────
CREATE TRIGGER book_fts_ai AFTER INSERT ON book BEGIN
  INSERT INTO book_fts (book_id, title) VALUES (new.id, COALESCE(new.title, ''));
END;
CREATE TRIGGER book_fts_ad AFTER DELETE ON book BEGIN
  DELETE FROM book_fts WHERE book_id = old.id;
END;
CREATE TRIGGER book_fts_au AFTER UPDATE OF title ON book BEGIN
  DELETE FROM book_fts WHERE book_id = old.id;
  INSERT INTO book_fts (book_id, title) VALUES (new.id, COALESCE(new.title, ''));
END;

-- ── Backfill existing rows ────────────────────────────────────────────
INSERT INTO series_fts (series_id, name) SELECT id, name FROM series;
INSERT INTO book_fts (book_id, title) SELECT id, COALESCE(title, '') FROM book;
