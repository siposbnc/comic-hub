-- 0008_bookmarks: per-user, per-book bookmarks — a saved place (page) with an optional
-- short note. Toggling the current page adds/removes a bookmark, so a page holds at most
-- one bookmark per user (the unique index enforces it). See docs/06-reader.md §6.

CREATE TABLE bookmark (
  id         TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  book_id    TEXT NOT NULL REFERENCES book(id) ON DELETE CASCADE,
  page       INTEGER NOT NULL,
  note       TEXT NOT NULL DEFAULT '',
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);

-- One bookmark per page (per user+book); also the lookup index for list-by-book.
CREATE UNIQUE INDEX idx_bookmark_user_book_page ON bookmark(user_id, book_id, page);
