-- Concurrent scans (e.g. the file-watcher's debounced scan overlapping an explicit one)
-- could create duplicate series rows for the same folder: unlike book.file_path, the
-- series (library_id, folder_path) pair had no unique constraint, and the scanner's
-- resolveSeries did an unguarded read-then-insert get-or-create. Merge any existing
-- duplicates, then enforce uniqueness so the scanner's ON CONFLICT upsert converges on a
-- single row instead of duplicating.

-- 1. Repoint books from non-canonical duplicate series onto the canonical (lowest id) row
--    for their (library_id, folder_path), so no books are lost when the dups are removed.
UPDATE book
SET series_id = (
	SELECT MIN(s2.id) FROM series s2
	JOIN series s1 ON s1.id = book.series_id
	WHERE s2.library_id = s1.library_id AND s2.folder_path = s1.folder_path
)
WHERE series_id IN (
	SELECT s1.id FROM series s1
	WHERE s1.folder_path IS NOT NULL
	  AND s1.id <> (
		SELECT MIN(s2.id) FROM series s2
		WHERE s2.library_id = s1.library_id AND s2.folder_path = s1.folder_path
	  )
);

-- 2. Delete the now-empty duplicate series (story_arc rows cascade via FK).
DELETE FROM series
WHERE folder_path IS NOT NULL
  AND id <> (
	SELECT MIN(s2.id) FROM series s2
	WHERE s2.library_id = series.library_id AND s2.folder_path = series.folder_path
  );

-- 3. Enforce uniqueness going forward (the scanner upserts ON CONFLICT of this index).
CREATE UNIQUE INDEX idx_series_library_folder ON series(library_id, folder_path);
