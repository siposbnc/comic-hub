-- 0018_book_ignored: let the user hide a mis-scanned or junk file from the library without
-- deleting it on disk. Ignored books drop out of every catalog view (series detail, grid
-- counts, tracker, search, recent) and are surfaced only under Library Health, where they
-- can be restored. Preserved across rescans (the book Upsert never rewrites this column).
ALTER TABLE book ADD COLUMN ignored INTEGER NOT NULL DEFAULT 0;
