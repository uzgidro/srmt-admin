-- Add type and count columns back to piezometers
ALTER TABLE piezometers ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'pressure';
ALTER TABLE piezometers ADD COLUMN count INT NOT NULL DEFAULT 0;

-- Recreate the index
CREATE INDEX IF NOT EXISTS idx_piezometers_org_type ON piezometers(organization_id, type);

-- Drop piezometer_counts table
DROP TABLE IF EXISTS piezometer_counts;
