-- 0011_settings: a small key/value store for app settings the user edits at runtime —
-- notably metadata-provider credentials (so they no longer have to be env-only and the
-- packaged app can set them). Values are stored verbatim; secrets never leave the server.

CREATE TABLE app_setting (
  key        TEXT PRIMARY KEY,
  value      TEXT NOT NULL,
  updated_at INTEGER NOT NULL
);
