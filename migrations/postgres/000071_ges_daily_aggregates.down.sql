DROP TRIGGER IF EXISTS ges_daily_data_check_aggregates_trg ON ges_daily_data;
DROP FUNCTION IF EXISTS ges_daily_data_check_aggregates();

ALTER TABLE ges_daily_data
    DROP COLUMN IF EXISTS modernization_aggregates,
    DROP COLUMN IF EXISTS repair_aggregates;
