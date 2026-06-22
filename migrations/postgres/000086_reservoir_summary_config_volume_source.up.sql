-- Adds volume_source to control how Volume.Current is resolved when
-- the daily snapshot from reservoir_data is missing or stale.
--
--   'static'       → existing behaviour: snapshot → level_volume curve
--                    → static.uz fallback. This is the default for every
--                    existing config so nothing breaks at deploy.
--   'level_volume' → curve takes priority over snapshot; even if a
--                    snapshot exists in reservoir_data, the level_volume
--                    interpolation result is used. Fall back to snapshot
--                    only when the curve is not configured for the org.
--
-- CHECK keeps the column closed: any other value is rejected at write,
-- the application surfaces a 400.

ALTER TABLE reservoir_summary_config
    ADD COLUMN volume_source VARCHAR(32) NOT NULL DEFAULT 'static'
    CHECK (volume_source IN ('static', 'level_volume'));
