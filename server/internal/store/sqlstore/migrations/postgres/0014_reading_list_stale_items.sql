-- 0014_reading_list_stale_items: reading lists survive book deletion. See the sqlite
-- migration of the same number for the full rationale.

CREATE TABLE reading_list_item_new (
  id           TEXT PRIMARY KEY,
  list_id      TEXT NOT NULL REFERENCES reading_list(id) ON DELETE CASCADE,
  book_id      TEXT REFERENCES book(id) ON DELETE SET NULL,
  position     DOUBLE PRECISION NOT NULL,
  added_at     BIGINT NOT NULL,
  series_name  TEXT NOT NULL DEFAULT '',
  number       TEXT NOT NULL DEFAULT '',
  title        TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL DEFAULT ''
);

INSERT INTO reading_list_item_new
  (id, list_id, book_id, position, added_at, series_name, number, title, content_hash)
SELECT md5(random()::text || clock_timestamp()::text || li.list_id || li.book_id),
       li.list_id, li.book_id, li.position, li.added_at,
       COALESCE(s.name, ''), COALESCE(b.number, ''), COALESCE(b.title, ''), COALESCE(b.content_hash, '')
FROM reading_list_item li
JOIN book b ON b.id = li.book_id
LEFT JOIN series s ON s.id = b.series_id;

DROP TABLE reading_list_item;
ALTER TABLE reading_list_item_new RENAME TO reading_list_item;

CREATE UNIQUE INDEX idx_reading_list_item_book ON reading_list_item(list_id, book_id)
  WHERE book_id IS NOT NULL;
CREATE INDEX idx_reading_list_item_stale_hash ON reading_list_item(content_hash)
  WHERE book_id IS NULL;
