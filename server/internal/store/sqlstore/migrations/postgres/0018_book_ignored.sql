-- 0018_book_ignored: hide a mis-scanned or junk file from the library without deleting it.
-- See the sqlite migration of the same number for the full rationale. 0/1 integer boolean
-- (kept as INTEGER to match the rest of the schema and the shared query filters).
ALTER TABLE book ADD COLUMN ignored INTEGER NOT NULL DEFAULT 0;
