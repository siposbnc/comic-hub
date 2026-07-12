-- 0015_tracker: the Tracker screen — a per-user reading matrix. See the sqlite migration
-- of the same number for the full rationale.

CREATE TABLE track (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);
CREATE INDEX idx_track_user ON track(user_id, name);

CREATE TABLE track_issue (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  track_id   TEXT REFERENCES track(id) ON DELETE CASCADE,
  series_id  TEXT REFERENCES series(id) ON DELETE CASCADE,
  number     TEXT NOT NULL,
  sort       DOUBLE PRECISION NOT NULL,
  is_read    INTEGER NOT NULL DEFAULT 0,
  read_at    BIGINT,
  created_at BIGINT NOT NULL
);
CREATE UNIQUE INDEX idx_track_issue_track_sort ON track_issue(track_id, sort) WHERE track_id IS NOT NULL;
CREATE UNIQUE INDEX idx_track_issue_series_sort ON track_issue(user_id, series_id, sort) WHERE series_id IS NOT NULL;
CREATE INDEX idx_track_issue_series ON track_issue(user_id, series_id);
