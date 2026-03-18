-- Allow multiple rows per organization (versioning)
ALTER TABLE reservoir_device_summary DROP CONSTRAINT IF EXISTS uq_reservoir_device_org;

-- Index for fast "latest version per org on a given date" queries
CREATE INDEX IF NOT EXISTS idx_reservoir_device_org_created
    ON reservoir_device_summary(organization_id, created_at DESC);
