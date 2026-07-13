-- 0016_book_kind: classify each book (issue / special / variant / cover). See the sqlite
-- migration of the same number for the full rationale.

ALTER TABLE book ADD COLUMN kind TEXT NOT NULL DEFAULT 'issue';
