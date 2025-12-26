-- Rollback: Remove total_income_volume columns from reservoir_data table

ALTER TABLE reservoir_data
    DROP COLUMN IF EXISTS total_income_volume_mln_m3;

ALTER TABLE reservoir_data
    DROP COLUMN IF EXISTS total_income_volume_prev_year_mln_m3;
