-- 0017_backfill_book_kind: classify books that predate 0016. See the sqlite migration of
-- the same number for the full rationale. The SQL is dialect-agnostic.

UPDATE book SET kind = 'annual'   WHERE kind = 'issue' AND lower(number) LIKE 'annual%';
UPDATE book SET kind = 'one-shot' WHERE kind = 'issue' AND (lower(number) LIKE 'one-shot%' OR lower(number) LIKE 'one shot%');
UPDATE book SET kind = 'tpb'      WHERE kind = 'issue' AND lower(number) LIKE 'tpb%';
UPDATE book SET kind = 'gn'       WHERE kind = 'issue' AND lower(number) LIKE 'gn%';
UPDATE book SET kind = 'special'  WHERE kind = 'issue' AND lower(number) LIKE 'special%';

UPDATE book SET kind = 'variant'  WHERE kind = 'issue' AND (
       lower(file_path) LIKE '%variant%'
    OR lower(file_path) LIKE '%(var)%'
    OR lower(file_path) LIKE '%cvr%'
    OR lower(file_path) LIKE '% var.%'
    OR lower(file_path) LIKE '% var %');

UPDATE book SET kind = 'cover'    WHERE kind = 'issue'
    AND (sort_number IS NULL OR sort_number = 0)
    AND lower(file_path) LIKE '%cover%';
