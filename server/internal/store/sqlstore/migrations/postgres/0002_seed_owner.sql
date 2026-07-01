-- 0002_seed_owner (Postgres dialect): the single implicit owner for embedded mode.
-- Mirrors migrations/sqlite/0002_seed_owner.sql.
INSERT INTO "user" (id, username, display_name, role, created_at)
VALUES ('owner', 'owner', 'Owner', 'owner', (extract(epoch FROM now()) * 1000)::bigint);
