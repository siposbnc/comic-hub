-- 0013_sessions: refresh-token sessions for server-mode auth (Phase 3 — Milestone A).
-- The `user` table already exists (0001) with password_hash/role/age_rating_max. This adds
-- the store for refresh tokens: we keep only the sha256 hash of each opaque token, so a DB
-- leak exposes nothing usable. Sessions are revocable (logout, password change) and pruned
-- past expiry. Access tokens are stateless JWTs and are not stored.

CREATE TABLE session (
  id           TEXT PRIMARY KEY,
  user_id      TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
  refresh_hash TEXT NOT NULL UNIQUE,
  expires_at   BIGINT NOT NULL,
  created_at   BIGINT NOT NULL
);

CREATE INDEX idx_session_user ON session(user_id);
CREATE INDEX idx_session_expires ON session(expires_at);
