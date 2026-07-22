-- 0019_reading_list_collection_ref: a reading list can reference a whole collection as a
-- single ordered entry. The reference occupies one position slot (like any item) and is
-- expanded — live — into the collection's current books when the list is read, rendered as
-- a named group. book_id stays NULL on a reference row (so the linked-book unique index is
-- untouched); collection_id points at the referenced collection. Deleting the collection
-- cascades the reference away, so a list never keeps a dangling group.

ALTER TABLE reading_list_item
  ADD COLUMN collection_id TEXT REFERENCES collection(id) ON DELETE CASCADE;

-- A collection appears at most once per list; reference rows are found by this partial index.
CREATE UNIQUE INDEX idx_reading_list_item_collection ON reading_list_item(list_id, collection_id)
  WHERE collection_id IS NOT NULL;
