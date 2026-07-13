-- 0017_backfill_book_kind: classify books that already existed before 0016 (a normal rescan
-- skips unchanged files, so their kind stays 'issue'). Uses the same signals the scanner
-- does, minus ComicInfo <Format> (which needs the archive) — the next rescan of a changed
-- file refines those. Only touches rows still at the default 'issue'.

UPDATE book SET kind = 'annual'   WHERE kind = 'issue' AND lower(number) LIKE 'annual%';
UPDATE book SET kind = 'one-shot' WHERE kind = 'issue' AND (lower(number) LIKE 'one-shot%' OR lower(number) LIKE 'one shot%');
UPDATE book SET kind = 'tpb'      WHERE kind = 'issue' AND lower(number) LIKE 'tpb%';
UPDATE book SET kind = 'gn'       WHERE kind = 'issue' AND lower(number) LIKE 'gn%';
UPDATE book SET kind = 'special'  WHERE kind = 'issue' AND lower(number) LIKE 'special%';

-- Variant art: markers that are safe even on a numbered file.
UPDATE book SET kind = 'variant'  WHERE kind = 'issue' AND (
       lower(file_path) LIKE '%variant%'
    OR lower(file_path) LIKE '%(var)%'
    OR lower(file_path) LIKE '%cvr%'
    OR lower(file_path) LIKE '% var.%'
    OR lower(file_path) LIKE '% var %');

-- Cover-only files: trust "cover" only when there's no resolvable issue number.
UPDATE book SET kind = 'cover'    WHERE kind = 'issue'
    AND (sort_number IS NULL OR sort_number = 0)
    AND lower(file_path) LIKE '%cover%';
