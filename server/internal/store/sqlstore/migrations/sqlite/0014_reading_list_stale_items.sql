-- 0014_reading_list_stale_items: reading lists survive book deletion. Items get their own
-- id, book_id becomes nullable (ON DELETE SET NULL instead of CASCADE), and every item
-- captures a display snapshot (series name, number, title, content hash) at add time.
-- A deleted book leaves a stale-but-visible placeholder that preserves the list's order;
-- content_hash lets a rescan of the same file auto-relink the placeholder to the
-- recreated book row. Manual placeholders (issues not in the library yet) are simply
-- rows born with a NULL book_id.

CREATE TABLE reading_list_item_new (
  id           TEXT PRIMARY KEY,
  list_id      TEXT NOT NULL REFERENCES reading_list(id) ON DELETE CASCADE,
  book_id      TEXT REFERENCES book(id) ON DELETE SET NULL,
  position     REAL NOT NULL,
  added_at     INTEGER NOT NULL,
  series_name  TEXT NOT NULL DEFAULT '',
  number       TEXT NOT NULL DEFAULT '',
  title        TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL DEFAULT ''
);

INSERT INTO reading_list_item_new
  (id, list_id, book_id, position, added_at, series_name, number, title, content_hash)
SELECT lower(hex(randomblob(16))), li.list_id, li.book_id, li.position, li.added_at,
       COALESCE(s.name, ''), COALESCE(b.number, ''), COALESCE(b.title, ''), COALESCE(b.content_hash, '')
FROM reading_list_item li
JOIN book b ON b.id = li.book_id
LEFT JOIN series s ON s.id = b.series_id;

DROP TABLE reading_list_item;
ALTER TABLE reading_list_item_new RENAME TO reading_list_item;

-- A linked book appears at most once per list; stale rows (NULL book_id) are unlimited.
CREATE UNIQUE INDEX idx_reading_list_item_book ON reading_list_item(list_id, book_id)
  WHERE book_id IS NOT NULL;
-- Fast auto-relink lookup: stale rows by the content hash of the book they pointed at.
CREATE INDEX idx_reading_list_item_stale_hash ON reading_list_item(content_hash)
  WHERE book_id IS NULL;
