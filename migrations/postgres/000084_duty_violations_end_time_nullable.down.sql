-- Rollback to "end_time required, end_time > start_time always".
-- WILL FAIL if any row has end_time IS NULL (intentional — the operator
-- must decide what to do with ongoing records before going back to the
-- old NOT NULL schema; an automatic backfill could silently destroy data).

ALTER TABLE duty_violations DROP CONSTRAINT duty_violations_time_range;
ALTER TABLE duty_violations
    ADD CONSTRAINT duty_violations_time_range
    CHECK (end_time > start_time);

ALTER TABLE duty_violations ALTER COLUMN end_time SET NOT NULL;
