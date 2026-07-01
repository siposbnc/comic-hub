-- 0006_reader_prefs: per-user, per-book reader overrides (layout, fit, direction, …) so a
-- book remembers how it was last read. The settings blob is opaque JSON owned by the
-- reader client; the server just stores and returns it. See docs/06-reader.md.

CREATE TABLE reader_pref (
  user_id    TEXT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
  book_id    TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  settings   TEXT NOT NULL DEFAULT '{}',
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (user_id, book_id)
);
