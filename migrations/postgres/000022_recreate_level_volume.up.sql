-- Drop the existing level_volume table
DROP TABLE IF EXISTS level_volume CASCADE;

-- Create new level_volume table with organization_id
CREATE TABLE level_volume (
    id BIGSERIAL PRIMARY KEY,
    level DOUBLE PRECISION NOT NULL,
    volume NUMERIC NOT NULL,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

-- Create unique constraint on organization_id, level, and volume
ALTER TABLE level_volume
    ADD CONSTRAINT level_volume_org_level_volume_unique UNIQUE (organization_id, level, volume);

-- Create index on organization_id for faster lookups
CREATE INDEX idx_level_volume_organization_id ON level_volume(organization_id);

-- Create trigger to automatically update updated_at timestamp
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON level_volume
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
