-- 0005_smart_lists: rule-based dynamic lists (docs/03-api.md §7). A smart list stores a
-- small JSON rule set; results are evaluated on demand against the catalog (and the acting
-- user's read state), never materialized. Shared like collections; owner_id records the
-- creator.

CREATE TABLE smart_list (
  id         TEXT PRIMARY KEY,
  owner_id   TEXT REFERENCES "user"(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  rules      TEXT NOT NULL DEFAULT '{}',
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);
