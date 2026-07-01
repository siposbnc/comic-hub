-- 0010_story_arcs: provider-sourced narrative arcs that span a series' issues. Populated by
-- an online match from each issue's story-arc credits; a re-match replaces them. Volumes are
-- not stored — they're derived from each book's `volume` field at read time. See docs/04-server.md.

CREATE TABLE story_arc (
  id          TEXT PRIMARY KEY,
  series_id   TEXT NOT NULL REFERENCES series(id) ON DELETE CASCADE,
  provider    TEXT NOT NULL DEFAULT '',
  provider_id TEXT NOT NULL DEFAULT '',
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  updated_at  INTEGER NOT NULL
);

CREATE INDEX idx_story_arc_series ON story_arc(series_id);

CREATE TABLE story_arc_book (
  arc_id  TEXT NOT NULL REFERENCES story_arc(id) ON DELETE CASCADE,
  book_id TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  PRIMARY KEY (arc_id, book_id)
);

CREATE INDEX idx_story_arc_book_book ON story_arc_book(book_id);
