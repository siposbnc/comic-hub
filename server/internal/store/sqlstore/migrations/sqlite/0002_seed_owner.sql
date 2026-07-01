-- 0002_seed_owner: the single implicit owner for embedded mode.
-- In embedded mode there are no accounts — a fixed-id "owner" row backs progress,
-- reading lists, etc. (the handshake endpoint reports this identity). Auth mode adds
-- real users alongside it in a later phase.
INSERT INTO user (id, username, display_name, role, created_at)
VALUES ('owner', 'owner', 'Owner', 'owner', CAST(strftime('%s', 'now') AS INTEGER) * 1000);
