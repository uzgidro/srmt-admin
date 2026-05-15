-- Rollback 000079.

DROP TRIGGER  IF EXISTS trg_reservoir_flood_hourly_delete_if_empty      ON reservoir_flood_hourly;
DROP TRIGGER  IF EXISTS trg_reservoir_flood_hourly_skip_insert_if_empty ON reservoir_flood_hourly;
DROP FUNCTION IF EXISTS reservoir_flood_hourly_delete_if_empty();
DROP FUNCTION IF EXISTS reservoir_flood_hourly_skip_insert_if_empty();
DROP FUNCTION IF EXISTS reservoir_flood_hourly_is_all_null(reservoir_flood_hourly);

-- Restore the dropped column. Existing rows get NULL (the field was never
-- populated meaningfully anyway).
ALTER TABLE reservoir_flood_hourly
    ADD COLUMN filtration_m3s NUMERIC CHECK (filtration_m3s >= 0);
