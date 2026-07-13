-- 0016_book_kind: classify each book as a normal issue, a special edition
-- (annual/one-shot/tpb/gn/special), or non-issue art (variant/cover). Populated at scan
-- time from ComicInfo <Format>, the parsed issue-number label, and filename heuristics.
-- The Tracker splits specials into their own rows and excludes variants/covers; existing
-- rows default to 'issue' until the next scan reclassifies them.

ALTER TABLE book ADD COLUMN kind TEXT NOT NULL DEFAULT 'issue';
