-- Add repair/modernization aggregates to ges_daily_data.
-- working + repair + modernization must not exceed ges_config.total_aggregates.
-- The upper-bound check needs a JOIN to ges_config, so it is enforced via trigger.
-- The non-negative bounds are simple column CHECKs.

ALTER TABLE ges_daily_data
    ADD COLUMN repair_aggregates        INT NOT NULL DEFAULT 0
        CONSTRAINT chk_ges_daily_data_repair_aggregates_nonneg CHECK (repair_aggregates >= 0),
    ADD COLUMN modernization_aggregates INT NOT NULL DEFAULT 0
        CONSTRAINT chk_ges_daily_data_modernization_aggregates_nonneg CHECK (modernization_aggregates >= 0);

CREATE OR REPLACE FUNCTION ges_daily_data_check_aggregates()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    cap INT;
BEGIN
    SELECT total_aggregates INTO cap
    FROM ges_config
    WHERE organization_id = NEW.organization_id;
    IF cap IS NULL THEN
        -- No ges_config row yet — allow the insert/update so data entry
        -- is not blocked before configuration is created.
        RETURN NEW;
    END IF;
    IF NEW.working_aggregates + NEW.repair_aggregates + NEW.modernization_aggregates > cap THEN
        RAISE EXCEPTION 'aggregates sum (%+%+%) exceeds total_aggregates (%) for org %',
            NEW.working_aggregates, NEW.repair_aggregates, NEW.modernization_aggregates,
            cap, NEW.organization_id
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END $$;

CREATE TRIGGER ges_daily_data_check_aggregates_trg
    BEFORE INSERT OR UPDATE ON ges_daily_data
    FOR EACH ROW EXECUTE FUNCTION ges_daily_data_check_aggregates();
