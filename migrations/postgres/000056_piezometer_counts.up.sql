-- Create piezometer_counts table (counts per organization, not per piezometer)
CREATE TABLE IF NOT EXISTS piezometer_counts (
    id                 BIGSERIAL PRIMARY KEY,
    organization_id    BIGINT NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    pressure_count     INT NOT NULL DEFAULT 0,
    non_pressure_count INT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id),
    updated_by_user_id BIGINT REFERENCES users(id)
);

-- Migrate existing data from piezometers into piezometer_counts
INSERT INTO piezometer_counts (organization_id, pressure_count, non_pressure_count)
SELECT
    organization_id,
    COALESCE(SUM(CASE WHEN type = 'pressure' THEN count ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN type = 'non_pressure' THEN count ELSE 0 END), 0)
FROM piezometers
GROUP BY organization_id
ON CONFLICT (organization_id) DO NOTHING;

-- Drop the index on (organization_id, type) since type column is being removed
DROP INDEX IF EXISTS idx_piezometers_org_type;

-- Remove type and count columns from piezometers
ALTER TABLE piezometers DROP COLUMN IF EXISTS type;
ALTER TABLE piezometers DROP COLUMN IF EXISTS count;
