-- Create receptions table
CREATE TABLE IF NOT EXISTS receptions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    date TIMESTAMPTZ NOT NULL,
    description TEXT,
    visitor VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'default' CHECK (status IN ('default', 'true', 'false')),

    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    created_by_user_id BIGINT NOT NULL,
    updated_by_user_id BIGINT,

    -- Foreign key constraints
    CONSTRAINT fk_receptions_created_by FOREIGN KEY (created_by_user_id)
        REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_receptions_updated_by FOREIGN KEY (updated_by_user_id)
        REFERENCES users(id) ON DELETE SET NULL
);

-- Create indexes for filtering and performance
CREATE INDEX IF NOT EXISTS idx_receptions_date ON receptions(date);
CREATE INDEX IF NOT EXISTS idx_receptions_status ON receptions(status);
CREATE INDEX IF NOT EXISTS idx_receptions_created_at ON receptions(created_at);
CREATE INDEX IF NOT EXISTS idx_receptions_created_by ON receptions(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_receptions_visitor ON receptions(visitor);

-- Create trigger for auto-updating updated_at timestamp
CREATE TRIGGER set_timestamp_receptions
    BEFORE UPDATE ON receptions
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Add comment to table
COMMENT ON TABLE receptions IS 'Reception records with visitor information';
COMMENT ON COLUMN receptions.status IS 'Reception status: default, true, or false';
