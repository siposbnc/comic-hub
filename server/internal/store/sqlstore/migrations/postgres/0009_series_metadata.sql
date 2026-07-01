-- 0009_series_metadata: series-level online-match state. metadata_state mirrors the
-- book-level states (none|matched|incomplete); match_provider/match_provider_id record the
-- linked provider volume so a re-match or story-arc/volume fetch reuses it. The scanner
-- never writes these columns, so a rescan preserves a series' match. See docs/04-server.md.

ALTER TABLE series ADD COLUMN metadata_state TEXT NOT NULL DEFAULT 'none';
ALTER TABLE series ADD COLUMN match_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE series ADD COLUMN match_provider_id TEXT NOT NULL DEFAULT '';
