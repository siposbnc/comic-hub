-- 0003_metadata (Postgres dialect): normalized people/genres/characters, per-field
-- locks, and per-book provider ids. Mirrors migrations/sqlite/0003_metadata.sql;
-- "character" is quoted (type keyword in Postgres).

ALTER TABLE book   ADD COLUMN provider_ids  TEXT NOT NULL DEFAULT '{}';
ALTER TABLE book   ADD COLUMN locked_fields TEXT NOT NULL DEFAULT '[]';
ALTER TABLE series ADD COLUMN locked_fields TEXT NOT NULL DEFAULT '[]';

-- ── Credits (writer/penciler/inker/…) ─────────────────────────────────
CREATE TABLE person (
  id   TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);
CREATE TABLE book_person (
  book_id   TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  person_id TEXT NOT NULL REFERENCES person(id) ON DELETE CASCADE,
  role      TEXT NOT NULL,
  PRIMARY KEY (book_id, person_id, role)
);
CREATE INDEX idx_book_person_book ON book_person(book_id);

-- ── Genres ────────────────────────────────────────────────────────────
CREATE TABLE genre (
  id   TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);
CREATE TABLE book_genre (
  book_id  TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  genre_id TEXT NOT NULL REFERENCES genre(id) ON DELETE CASCADE,
  PRIMARY KEY (book_id, genre_id)
);

-- ── Characters ────────────────────────────────────────────────────────
CREATE TABLE "character" (
  id   TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE
);
CREATE TABLE book_character (
  book_id      TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  character_id TEXT NOT NULL REFERENCES "character"(id) ON DELETE CASCADE,
  PRIMARY KEY (book_id, character_id)
);
