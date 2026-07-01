# 02 — Data Model

The catalog lives in **SQLite (WAL)** by default; the same schema runs on Postgres for
large remote installs. All access goes through a `Repository` interface so the domain is
storage-agnostic. IDs are **ULIDs** (sortable, URL-safe, no central coordination).

## 1. Entity overview

```
Library 1───* Series 1───* Book 1───* Page
                 │            │
                 │            ├──* BookMetadata(person/genre/tag/character links)
                 │            └──* ReadProgress *───1 User
                 │
Collection *──────┘ (Book ⇄ Collection via collection_item, ordered)
ReadingList 1───* reading_list_item *───1 Book   (per-user, ordered)
SmartList (rule JSON) ─ evaluated at query time
User 1───* Session / ApiToken
Tag, Person, Genre, Character, Publisher  ── many-to-many with Book
Job ── background work records
```

### Core relationships

- A **Book** belongs to exactly one **Series** (a synthetic "Standalone" series exists per
  library for loose files).
- A **Series** belongs to exactly one **Library**.
- **Progress** is per `(user, book)`.
- **Collections** and **Reading Lists** are ordered many-to-many over Books.
- **People** (writer, artist, etc.), **Genres**, **Characters**, **Tags**, **Publishers**
  are normalized and linked many-to-many with role qualifiers where relevant.

## 2. Schema (SQLite dialect)

```sql
-- ─────────────────────────── Libraries & files ───────────────────────────
CREATE TABLE library (
  id            TEXT PRIMARY KEY,             -- ULID
  name          TEXT NOT NULL,
  kind          TEXT NOT NULL DEFAULT 'comic',-- comic | manga (affects defaults: RTL, naming)
  created_at    INTEGER NOT NULL,             -- epoch ms
  updated_at    INTEGER NOT NULL,
  scan_options  TEXT NOT NULL DEFAULT '{}'    -- JSON: provider prefs, naming rules, etc.
);

CREATE TABLE library_root (                   -- a library may have multiple root folders
  id            TEXT PRIMARY KEY,
  library_id    TEXT NOT NULL REFERENCES library(id) ON DELETE CASCADE,
  path          TEXT NOT NULL,                -- absolute, normalized
  enabled       INTEGER NOT NULL DEFAULT 1,
  UNIQUE(library_id, path)
);

CREATE TABLE series (
  id            TEXT PRIMARY KEY,
  library_id    TEXT NOT NULL REFERENCES library(id) ON DELETE CASCADE,
  folder_path   TEXT,                          -- nullable for synthetic/standalone
  name          TEXT NOT NULL,
  sort_name     TEXT NOT NULL,                 -- "Batman, The" style sort key
  year          INTEGER,
  publisher_id  TEXT REFERENCES publisher(id),
  description   TEXT,
  reading_dir   TEXT NOT NULL DEFAULT 'ltr',   -- ltr | rtl  (manga default rtl)
  cover_book_id TEXT,                           -- which book supplies the series cover
  provider_ids  TEXT NOT NULL DEFAULT '{}',     -- JSON {comicvine:"4050-...", ...}
  total_count   INTEGER,                        -- expected # issues (from provider), nullable
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL
);
CREATE INDEX idx_series_library ON series(library_id);
CREATE INDEX idx_series_sort ON series(library_id, sort_name);

CREATE TABLE book (
  id            TEXT PRIMARY KEY,
  series_id     TEXT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
  library_id    TEXT NOT NULL REFERENCES library(id) ON DELETE CASCADE,
  file_path     TEXT NOT NULL UNIQUE,           -- absolute path on server
  file_format   TEXT NOT NULL,                  -- cbz|cbr|cb7|cbt|pdf|epub
  file_size     INTEGER NOT NULL,
  file_mtime    INTEGER NOT NULL,
  content_hash  TEXT,                            -- xxhash64 of file (for dedup + reader sync)
  page_count    INTEGER NOT NULL DEFAULT 0,
  -- denormalized metadata for fast listing/sorting (source of truth in metadata tables)
  title         TEXT,
  number        TEXT,                            -- "1", "1.MU", "Annual 2" (string!)
  sort_number   REAL,                            -- numeric sort key derived from number
  volume        INTEGER,
  release_date  INTEGER,                         -- epoch ms, nullable
  age_rating    TEXT,                            -- "Everyone" | "Teen" | "Mature" | ...
  language      TEXT,                            -- BCP-47
  summary       TEXT,
  cover_page    INTEGER NOT NULL DEFAULT 0,      -- which page index is the cover
  metadata_state TEXT NOT NULL DEFAULT 'none',   -- none|sidecar|matched|locked
  is_corrupt    INTEGER NOT NULL DEFAULT 0,
  added_at      INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL
);
CREATE INDEX idx_book_series ON book(series_id, sort_number);
CREATE INDEX idx_book_library ON book(library_id);
CREATE INDEX idx_book_hash ON book(content_hash);
CREATE INDEX idx_book_added ON book(library_id, added_at DESC);

CREATE TABLE page (
  book_id       TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  idx           INTEGER NOT NULL,               -- 0-based order
  file_name     TEXT NOT NULL,                  -- entry name inside the archive
  width         INTEGER,
  height        INTEGER,
  size          INTEGER,
  page_type     TEXT,                           -- FrontCover|Story|Advertisement|... (ComicInfo)
  is_double     INTEGER NOT NULL DEFAULT 0,     -- wide spread
  PRIMARY KEY (book_id, idx)
);

-- ─────────────────────────── Metadata (normalized) ───────────────────────────
CREATE TABLE publisher  (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE);
CREATE TABLE person     (id TEXT PRIMARY KEY, name TEXT NOT NULL, sort_name TEXT NOT NULL);
CREATE TABLE genre      (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE);
CREATE TABLE character  (id TEXT PRIMARY KEY, name TEXT NOT NULL);
CREATE TABLE tag        (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, color TEXT);

CREATE TABLE book_person (
  book_id   TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  person_id TEXT NOT NULL REFERENCES person(id) ON DELETE CASCADE,
  role      TEXT NOT NULL,    -- writer|penciller|inker|colorist|letterer|cover|editor
  PRIMARY KEY (book_id, person_id, role)
);
CREATE TABLE book_genre     (book_id TEXT, genre_id TEXT, PRIMARY KEY(book_id,genre_id));
CREATE TABLE book_character (book_id TEXT, character_id TEXT, PRIMARY KEY(book_id,character_id));
CREATE TABLE book_tag       (book_id TEXT, tag_id TEXT, PRIMARY KEY(book_id,tag_id));

-- ─────────────────────────── Users & auth ───────────────────────────
CREATE TABLE user (
  id            TEXT PRIMARY KEY,
  username      TEXT NOT NULL UNIQUE,
  display_name  TEXT NOT NULL,
  role          TEXT NOT NULL DEFAULT 'member', -- owner|admin|member|restricted
  password_hash TEXT,                            -- argon2id; null in embedded mode
  age_rating_max TEXT,                           -- ceiling for restricted users
  prefs         TEXT NOT NULL DEFAULT '{}',      -- JSON: reader defaults, theme, etc.
  created_at    INTEGER NOT NULL
);

CREATE TABLE api_token (
  id          TEXT PRIMARY KEY,
  user_id     TEXT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  token_hash  TEXT NOT NULL,        -- sha-256 of token
  scopes      TEXT NOT NULL DEFAULT '["read"]',
  last_used   INTEGER,
  created_at  INTEGER NOT NULL,
  expires_at  INTEGER
);

-- ─────────────────────────── Reading & progress ───────────────────────────
CREATE TABLE read_progress (
  user_id       TEXT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
  book_id       TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  page          INTEGER NOT NULL DEFAULT 0,      -- current page index
  page_count    INTEGER NOT NULL,                -- snapshot for % even if file changes
  status        TEXT NOT NULL DEFAULT 'unread',  -- unread|in_progress|read
  started_at    INTEGER,
  finished_at   INTEGER,
  updated_at    INTEGER NOT NULL,
  device        TEXT,                            -- last device that wrote progress
  PRIMARY KEY (user_id, book_id)
);
CREATE INDEX idx_progress_continue ON read_progress(user_id, status, updated_at DESC);

CREATE TABLE read_history (                       -- append-only reading events (stats)
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL,
  book_id    TEXT NOT NULL,
  event      TEXT NOT NULL,    -- opened|page_turn|finished
  page       INTEGER,
  at         INTEGER NOT NULL
);
CREATE INDEX idx_history_user ON read_history(user_id, at DESC);

-- ─────────────────────────── Collections & lists ───────────────────────────
CREATE TABLE collection (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  description TEXT,
  cover_book_id TEXT,
  owner_id    TEXT REFERENCES user(id),          -- null = shared/global
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL
);
CREATE TABLE collection_item (
  collection_id TEXT NOT NULL REFERENCES collection(id) ON DELETE CASCADE,
  book_id       TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  position      REAL NOT NULL,                    -- fractional index for cheap reordering
  PRIMARY KEY (collection_id, book_id)
);

CREATE TABLE reading_list (
  id          TEXT PRIMARY KEY,
  user_id     TEXT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL
);
CREATE TABLE reading_list_item (
  list_id   TEXT NOT NULL REFERENCES reading_list(id) ON DELETE CASCADE,
  book_id   TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  position  REAL NOT NULL,
  added_at  INTEGER NOT NULL,
  PRIMARY KEY (list_id, book_id)
);

CREATE TABLE smart_list (
  id          TEXT PRIMARY KEY,
  owner_id    TEXT REFERENCES user(id),
  name        TEXT NOT NULL,
  rules       TEXT NOT NULL,    -- JSON rule tree (see §4)
  sort        TEXT NOT NULL DEFAULT 'added_desc',
  created_at  INTEGER NOT NULL
);

-- ─────────────────────────── Jobs ───────────────────────────
CREATE TABLE job (
  id          TEXT PRIMARY KEY,
  type        TEXT NOT NULL,    -- scan|thumbnail|metadata_match|watch|organize|export
  state       TEXT NOT NULL,    -- queued|running|done|failed|canceled
  payload     TEXT NOT NULL,    -- JSON
  progress    REAL NOT NULL DEFAULT 0,
  total       INTEGER,
  done        INTEGER NOT NULL DEFAULT 0,
  error       TEXT,
  created_at  INTEGER NOT NULL,
  started_at  INTEGER,
  finished_at INTEGER
);
CREATE INDEX idx_job_state ON job(state, created_at);
```

## 3. Full-text search

```sql
CREATE VIRTUAL TABLE book_fts USING fts5(
  title, series_name, number, summary, people, characters, tags,
  content='', tokenize='unicode61 remove_diacritics 2'
);
```

- Populated/maintained by triggers (or by the indexer job after metadata changes).
- Search ranks by BM25 with field boosts (title > series > people > summary).
- A parallel `series_fts` supports series-level search.

## 4. Smart list rule format

Rules are a serializable boolean tree evaluated into a SQL `WHERE` clause (with a safe,
whitelisted field/operator map — never raw SQL from clients).

```jsonc
{
  "match": "all", // all | any
  "rules": [
    { "field": "library", "op": "is", "value": "DC" },
    { "field": "status", "op": "is_not", "value": "read" },
    { "field": "year", "op": "gte", "value": 2024 },
    {
      "match": "any",
      "rules": [
        { "field": "genre", "op": "contains", "value": "Cosmic" },
        { "field": "tag", "op": "contains", "value": "Crossover" },
      ],
    },
  ],
}
```

Whitelisted fields: `library, series, publisher, genre, tag, character, person, year,
status, age_rating, added_at, release_date, page_count, format, language, reading_dir`.
Operators per field type: `is/is_not, contains/not_contains, gte/lte/between, before/after`.

## 5. Key derivation & invariants

- **`sort_number`** is parsed from the messy `number` string: leading numeric → real;
  specials (`Annual`, `.MU`, letters) sort after with deterministic tie-breaks. Documented
  in [04-server.md](04-server.md) §metadata.
- **`content_hash`** (xxhash64 over the file bytes, or a fast sampled hash for huge files)
  powers dedup detection and reader↔server progress reconciliation.
- **Denormalized fields on `book`** (`title`, `number`, `release_date`, …) are a read cache;
  the metadata tables are authoritative. A single writer path keeps them consistent.
- **`metadata_state`** state machine: `none → sidecar → matched → locked`. `locked` means
  user-edited; scrapers must not overwrite it.
- Deleting a file on disk does **not** immediately delete the `book` row — it's marked
  `missing` (a flag in a later migration) so progress/lists survive a moved drive; a GC job
  prunes long-missing rows on user confirmation.

## 6. Migrations

- Versioned, forward-only SQL migrations embedded in the binary (e.g. `golang-migrate` or
  a tiny in-house runner). `schema_version` row gates startup; the server auto-migrates on
  launch and refuses to run against a newer schema than it knows.
