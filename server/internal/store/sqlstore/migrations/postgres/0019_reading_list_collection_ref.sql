-- 0019_reading_list_collection_ref: a reading list can reference a whole collection as a
-- single ordered entry. See the sqlite migration of the same number for the full rationale.

ALTER TABLE reading_list_item
  ADD COLUMN collection_id TEXT REFERENCES collection(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX idx_reading_list_item_collection ON reading_list_item(list_id, collection_id)
  WHERE collection_id IS NOT NULL;
