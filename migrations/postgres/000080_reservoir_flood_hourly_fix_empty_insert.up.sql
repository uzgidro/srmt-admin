-- 000080: fix partial-update clear broken by the BEFORE INSERT trigger.
--
-- Migration 000079 added a BEFORE INSERT trigger that returns NULL for a
-- fully-empty row. But UpsertReservoirFloodHourly clears a field via
-- INSERT ... ON CONFLICT DO UPDATE, and its INSERT VALUES carry .Value for
-- ALL columns (Set flags only drive the DO UPDATE CASE WHENs). A request
-- that clears one field — and names no numeric field — produces an
-- all-NULL INSERT row. The BEFORE INSERT trigger cancels that insert
-- BEFORE Postgres detects the conflict, so ON CONFLICT DO UPDATE never
-- runs and the existing row is left untouched.
--
-- Fix: move the empty-row pruning from BEFORE INSERT to AFTER INSERT.
-- AFTER INSERT fires only once the INSERT-or-upsert has fully settled, so
-- it no longer suppresses the DO UPDATE branch; it just deletes a row that
-- a genuine empty insert left behind.

-- 1. Remove the BEFORE INSERT trigger and its function.
DROP TRIGGER IF EXISTS trg_reservoir_flood_hourly_skip_insert_if_empty
    ON reservoir_flood_hourly;
DROP FUNCTION IF EXISTS reservoir_flood_hourly_skip_insert_if_empty();

-- 2. AFTER INSERT: if the freshly inserted row is fully empty, delete it.
--    Deleting NEW.id inside an AFTER INSERT row trigger is safe — the row
--    is already visible within the transaction, and there is no DELETE
--    trigger on this table to cascade into.
CREATE OR REPLACE FUNCTION reservoir_flood_hourly_delete_insert_if_empty()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    IF reservoir_flood_hourly_is_all_null(NEW) THEN
        DELETE FROM reservoir_flood_hourly WHERE id = NEW.id;
    END IF;
    RETURN NULL;
END
$$;

CREATE TRIGGER trg_reservoir_flood_hourly_delete_insert_if_empty
AFTER INSERT ON reservoir_flood_hourly
FOR EACH ROW EXECUTE FUNCTION reservoir_flood_hourly_delete_insert_if_empty();

-- 3. One-shot cleanup of any all-NULL ghost rows accrued since 000079.
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
