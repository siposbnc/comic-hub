-- 0007_reading_list_active: a user can mark one reading list "active" — the ordered queue
-- the reader advances through and the Home screen surfaces "next up" from. Activeness is
-- mutually exclusive per user (enforced by the service in a transaction).

ALTER TABLE reading_list ADD COLUMN is_active INTEGER NOT NULL DEFAULT 0;
