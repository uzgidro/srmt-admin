DROP INDEX IF EXISTS idx_piezometer_measurements_comparison_date;
DROP INDEX IF EXISTS idx_filtration_measurements_comparison_date;
ALTER TABLE filtration_measurements DROP COLUMN IF EXISTS comparison_date;
ALTER TABLE piezometer_measurements DROP COLUMN IF EXISTS comparison_date;
