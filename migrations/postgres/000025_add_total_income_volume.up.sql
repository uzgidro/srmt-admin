-- Add total_income_volume_mln_m3 columns to reservoir_data table
-- These columns allow manual entry of total income volume values
-- When NULL or 0, the system will calculate from income_m3_s data

ALTER TABLE reservoir_data
    ADD COLUMN total_income_volume_mln_m3 NUMERIC DEFAULT NULL;

ALTER TABLE reservoir_data
    ADD COLUMN total_income_volume_prev_year_mln_m3 NUMERIC DEFAULT NULL;

COMMENT ON COLUMN reservoir_data.total_income_volume_mln_m3 IS
    'Manually entered total income volume for current year (million m³). When NULL or 0, system calculates from income_m3_s.';

COMMENT ON COLUMN reservoir_data.total_income_volume_prev_year_mln_m3 IS
    'Manually entered total income volume for previous year (million m³). When NULL or 0, system calculates from income_m3_s.';
