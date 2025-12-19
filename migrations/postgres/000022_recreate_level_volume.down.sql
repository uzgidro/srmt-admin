-- Drop the new level_volume table
DROP TABLE IF EXISTS level_volume CASCADE;

-- Restore the original level_volume table structure
CREATE TABLE IF NOT EXISTS level_volume (
    id BIGSERIAL PRIMARY KEY,
    level DOUBLE PRECISION NOT NULL,
    volume DOUBLE PRECISION NOT NULL,
    res_id BIGINT NOT NULL REFERENCES reservoirs(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

-- Restore trigger for updated_at timestamp
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON level_volume
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
