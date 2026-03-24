ALTER TABLE filtration_measurements ADD COLUMN comparison_date DATE;
ALTER TABLE piezometer_measurements ADD COLUMN comparison_date DATE;

-- Partial indexes for GetComparisonDates / GetComparisonDatesBatch queries.
-- Matches the JOIN path: filtration_locations(organization_id) → filtration_measurements(location_id, date)
-- Only indexes rows that have a comparison_date set, keeping the index small.
CREATE INDEX idx_filtration_measurements_comparison_date
    ON filtration_measurements(location_id, date)
    WHERE comparison_date IS NOT NULL;

CREATE INDEX idx_piezometer_measurements_comparison_date
    ON piezometer_measurements(piezometer_id, date)
    WHERE comparison_date IS NOT NULL;
