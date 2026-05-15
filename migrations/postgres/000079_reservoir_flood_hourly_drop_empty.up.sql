-- 000079: drop unused filtration_m3s column + auto-prune empty rows.
--
-- Background: reservoir_flood_hourly accumulates "ghost" rows where every
-- data field is NULL. The UI's "clear field" action sends a PATCH with
-- explicit nulls, which UpsertReservoirFloodHourly stores verbatim via the
-- CASE WHEN $Set THEN EXCLUDED.col ELSE existing partial-update pattern.
-- A row whose 10 data fields all became NULL still sits in the table and
-- is picked up as "latest before T" by the sel-export prev-snapshot query,
-- producing blank prev cells in "Тезкор маълумот".
--
-- Two changes in one migration because they touch the same table and the
-- triggers must reference the post-migration column set:
--   1) DROP filtration_m3s — the column was never wired into the report
--      and is no longer needed by any caller.
--   2) Auto-prune ghost rows: BEFORE INSERT cancels a fully-empty insert,
--      AFTER UPDATE deletes a row that the update emptied. Plus a one-shot
--      DELETE for ghosts already in the table.

-- 1. Drop the filtration column. Forward-only — restored by the .down.sql.
ALTER TABLE reservoir_flood_hourly DROP COLUMN filtration_m3s;

-- 2. One-shot cleanup of existing ghost rows. Done BEFORE the trigger is
--    created so the DELETE runs as a plain bulk operation (no per-row
--    trigger overhead) and so the trigger only sees a clean table going
--    forward.
DELETE FROM reservoir_flood_hourly
 WHERE water_level_m       IS NULL
   AND water_volume_mln_m3 IS NULL
   AND inflow_m3s          IS NULL
   AND outflow_m3s         IS NULL
   AND ges_flow_m3s        IS NULL
   AND idle_discharge_m3s  IS NULL
   AND capacity_mwt        IS NULL
   AND duty_name           IS NULL
   AND weather_condition   IS NULL
   AND temperature_c       IS NULL;

-- 3. Predicate: TRUE iff every data column on the row is NULL.
--    Marked IMMUTABLE because the result depends only on the row's columns.
CREATE OR REPLACE FUNCTION reservoir_flood_hourly_is_all_null(r reservoir_flood_hourly)
RETURNS BOOLEAN
LANGUAGE SQL IMMUTABLE AS $$
    SELECT r.water_level_m       IS NULL
       AND r.water_volume_mln_m3 IS NULL
       AND r.inflow_m3s          IS NULL
       AND r.outflow_m3s         IS NULL
       AND r.ges_flow_m3s        IS NULL
       AND r.idle_discharge_m3s  IS NULL
       AND r.capacity_mwt        IS NULL
       AND r.duty_name           IS NULL
       AND r.weather_condition   IS NULL
       AND r.temperature_c       IS NULL
$$;

-- 4. BEFORE INSERT: silently drop a fully-empty insert. Returning NULL
--    from a row-level BEFORE INSERT trigger cancels the insert without
--    raising. This is fine for ON CONFLICT DO UPDATE upserts: when the
--    insert is cancelled, no conflict is reached and the upsert is a no-op.
CREATE OR REPLACE FUNCTION reservoir_flood_hourly_skip_insert_if_empty()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    IF reservoir_flood_hourly_is_all_null(NEW) THEN
        RETURN NULL;
    END IF;
    RETURN NEW;
END
$$;

CREATE TRIGGER trg_reservoir_flood_hourly_skip_insert_if_empty
BEFORE INSERT ON reservoir_flood_hourly
FOR EACH ROW EXECUTE FUNCTION reservoir_flood_hourly_skip_insert_if_empty();

-- 5. AFTER UPDATE: if the update result is fully-empty, delete the row.
--    AFTER (not BEFORE) so any ON CONFLICT DO UPDATE finishes its writes
--    and the row reaches its final state before we test the predicate.
--    The DELETE here will fire its own row trigger if/when one exists,
--    but we have no DELETE trigger, so this is safe.
CREATE OR REPLACE FUNCTION reservoir_flood_hourly_delete_if_empty()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    IF reservoir_flood_hourly_is_all_null(NEW) THEN
        DELETE FROM reservoir_flood_hourly WHERE id = NEW.id;
    END IF;
    RETURN NULL;
END
$$;

CREATE TRIGGER trg_reservoir_flood_hourly_delete_if_empty
AFTER UPDATE ON reservoir_flood_hourly
FOR EACH ROW EXECUTE FUNCTION reservoir_flood_hourly_delete_if_empty();
