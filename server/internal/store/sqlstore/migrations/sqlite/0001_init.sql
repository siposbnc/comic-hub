-- 0001_init: core catalog, users, progress, lists, and jobs.
-- Full metadata link tables (persons/genres/characters) and FTS arrive in later
-- migrations (Phase 1/2). See docs/02-data-model.md for the complete model.

-- ── Libraries & files ────────────────────────────────────────────────
CREATE TABLE library (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  kind          TEXT NOT NULL DEFAULT 'comic',
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL,
  scan_options  TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE library_root (
  id            TEXT PRIMARY KEY,
  library_id    TEXT NOT NULL REFERENCES library(id) ON DELETE CASCADE,
  path          TEXT NOT NULL,
  enabled       INTEGER NOT NULL DEFAULT 1,
  UNIQUE(library_id, path)
);

CREATE TABLE series (
  id            TEXT PRIMARY KEY,
  library_id    TEXT NOT NULL REFERENCES library(id) ON DELETE CASCADE,
  folder_path   TEXT,
  name          TEXT NOT NULL,
  sort_name     TEXT NOT NULL,
  year          INTEGER,
  publisher     TEXT,
  description   TEXT,
  reading_dir   TEXT NOT NULL DEFAULT 'ltr',
  cover_book_id TEXT,
  provider_ids  TEXT NOT NULL DEFAULT '{}',
  total_count   INTEGER,
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL
);
CREATE INDEX idx_series_library ON series(library_id);
CREATE INDEX idx_series_sort ON series(library_id, sort_name);

CREATE TABLE book (
  id             TEXT PRIMARY KEY,
  series_id      TEXT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
  library_id     TEXT NOT NULL REFERENCES library(id) ON DELETE CASCADE,
  file_path      TEXT NOT NULL UNIQUE,
  file_format    TEXT NOT NULL,
  file_size      INTEGER NOT NULL,
  file_mtime     INTEGER NOT NULL,
  content_hash   TEXT,
  page_count     INTEGER NOT NULL DEFAULT 0,
  title          TEXT,
  number         TEXT,
  sort_number    REAL,
  volume         INTEGER,
  release_date   INTEGER,
  age_rating     TEXT,
  language       TEXT,
  summary        TEXT,
  cover_page     INTEGER NOT NULL DEFAULT 0,
  metadata_state TEXT NOT NULL DEFAULT 'none',
  is_corrupt     INTEGER NOT NULL DEFAULT 0,
  added_at       INTEGER NOT NULL,
  updated_at     INTEGER NOT NULL
);
CREATE INDEX idx_book_series ON book(series_id, sort_number);
CREATE INDEX idx_book_library ON book(library_id);
CREATE INDEX idx_book_hash ON book(content_hash);
CREATE INDEX idx_book_added ON book(library_id, added_at DESC);

CREATE TABLE page (
  book_id    TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  idx        INTEGER NOT NULL,
  file_name  TEXT NOT NULL,
  width      INTEGER,
  height     INTEGER,
  size       INTEGER,
  page_type  TEXT,
  is_double  INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (book_id, idx)
);

-- ── Tags (first metadata facet; persons/genres/characters added later) ──
CREATE TABLE tag (
  id    TEXT PRIMARY KEY,
  name  TEXT NOT NULL UNIQUE,
  color TEXT
);
CREATE TABLE book_tag (
  book_id TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  tag_id  TEXT NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
  PRIMARY KEY (book_id, tag_id)
);

-- ── Users & auth ──────────────────────────────────────────────────────
CREATE TABLE user (
  id             TEXT PRIMARY KEY,
  username       TEXT NOT NULL UNIQUE,
  display_name   TEXT NOT NULL,
  role           TEXT NOT NULL DEFAULT 'member',
  password_hash  TEXT,
  age_rating_max TEXT,
  prefs          TEXT NOT NULL DEFAULT '{}',
  created_at     INTEGER NOT NULL
);

-- ── Reading progress & history ────────────────────────────────────────
CREATE TABLE read_progress (
  user_id     TEXT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
  book_id     TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  page        INTEGER NOT NULL DEFAULT 0,
  page_count  INTEGER NOT NULL,
  status      TEXT NOT NULL DEFAULT 'unread',
  started_at  INTEGER,
  finished_at INTEGER,
  updated_at  INTEGER NOT NULL,
  device      TEXT,
  PRIMARY KEY (user_id, book_id)
);
CREATE INDEX idx_progress_continue ON read_progress(user_id, status, updated_at DESC);

CREATE TABLE read_history (
  id      TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  book_id TEXT NOT NULL,
  event   TEXT NOT NULL,
  page    INTEGER,
  at      INTEGER NOT NULL
);
CREATE INDEX idx_history_user ON read_history(user_id, at DESC);

-- ── Collections & reading lists ───────────────────────────────────────
CREATE TABLE collection (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  description   TEXT,
  cover_book_id TEXT,
  owner_id      TEXT REFERENCES user(id),
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL
);
CREATE TABLE collection_item (
  collection_id TEXT NOT NULL REFERENCES collection(id) ON DELETE CASCADE,
  book_id       TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  position      REAL NOT NULL,
  PRIMARY KEY (collection_id, book_id)
);

CREATE TABLE reading_list (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE reading_list_item (
  list_id  TEXT NOT NULL REFERENCES reading_list(id) ON DELETE CASCADE,
  book_id  TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  position REAL NOT NULL,
  added_at INTEGER NOT NULL,
  PRIMARY KEY (list_id, book_id)
);

-- ── Background jobs ───────────────────────────────────────────────────
CREATE TABLE job (
  id          TEXT PRIMARY KEY,
  type        TEXT NOT NULL,
  state       TEXT NOT NULL,
  payload     TEXT NOT NULL,
  progress    REAL NOT NULL DEFAULT 0,
  total       INTEGER,
  done        INTEGER NOT NULL DEFAULT 0,
  error       TEXT,
  created_at  INTEGER NOT NULL,
  started_at  INTEGER,
  finished_at INTEGER
);
CREATE INDEX idx_job_state ON job(state, created_at);
