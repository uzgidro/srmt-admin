-- Revert 000080: restore the BEFORE INSERT trigger from 000079.
--
-- Note: this reinstates the partial-update clear bug. The forward
-- migration is the correct state; this down path exists only for
-- migration symmetry.

DROP TRIGGER IF EXISTS trg_reservoir_flood_hourly_delete_insert_if_empty
    ON reservoir_flood_hourly;
DROP FUNCTION IF EXISTS reservoir_flood_hourly_delete_insert_if_empty();

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
