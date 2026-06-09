-- Make end_time optional so an in-progress (ongoing) duty violation can be
-- recorded the moment the absence is noticed, with end_time filled in
-- later via PATCH once the situation is resolved.

ALTER TABLE duty_violations ALTER COLUMN end_time DROP NOT NULL;

-- Old CHECK required end_time > start_time unconditionally. Replace with
-- the same predicate, gated on end_time being present.
ALTER TABLE duty_violations DROP CONSTRAINT duty_violations_time_range;
ALTER TABLE duty_violations
    ADD CONSTRAINT duty_violations_time_range
    CHECK (end_time IS NULL OR end_time > start_time);
