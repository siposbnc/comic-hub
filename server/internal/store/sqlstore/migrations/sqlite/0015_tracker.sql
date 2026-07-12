-- 0015_tracker: the Tracker screen — a per-user reading matrix over every series.
-- Library series and their issues are projected live from the catalog (series/book/
-- read_progress); only the user's *additions* are stored here:
--   * a standalone track — a series the user tracks that is in no library, and
--   * an overlay issue — an issue with no backing file, attached either to a library
--     series (a "gap" the user wants tracked) or to a standalone track. Its read flag
--     is independent of any book, so an issue read elsewhere can still be marked read.
-- Library issues keep their read state in read_progress (via /mark), never here.

CREATE TABLE track (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX idx_track_user ON track(user_id, name);

CREATE TABLE track_issue (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
  track_id   TEXT REFERENCES track(id) ON DELETE CASCADE,   -- set for a standalone-track issue
  series_id  TEXT REFERENCES series(id) ON DELETE CASCADE,  -- set for an overlay on a library series
  number     TEXT NOT NULL,
  sort       REAL NOT NULL,
  is_read    INTEGER NOT NULL DEFAULT 0,
  read_at    INTEGER,
  created_at INTEGER NOT NULL
);
-- One overlay issue per number, within a standalone track or within a user's view of a series.
CREATE UNIQUE INDEX idx_track_issue_track_sort ON track_issue(track_id, sort) WHERE track_id IS NOT NULL;
CREATE UNIQUE INDEX idx_track_issue_series_sort ON track_issue(user_id, series_id, sort) WHERE series_id IS NOT NULL;
CREATE INDEX idx_track_issue_series ON track_issue(user_id, series_id);
