-- Create fast_calls table
CREATE TABLE IF NOT EXISTS fast_calls (
    id BIGSERIAL PRIMARY KEY,
    contact_id BIGINT NOT NULL,
    position INTEGER NOT NULL,

    -- Foreign key constraint
    CONSTRAINT fk_fast_calls_contact FOREIGN KEY (contact_id)
        REFERENCES contacts(id) ON DELETE CASCADE,

    -- Ensure unique position values
    CONSTRAINT uq_fast_calls_position UNIQUE (position)
);

-- Create index on contact_id for faster lookups
CREATE INDEX IF NOT EXISTS idx_fast_calls_contact_id ON fast_calls(contact_id);

-- Create index on position for ordering
CREATE INDEX IF NOT EXISTS idx_fast_calls_position ON fast_calls(position);

-- Add comment to table
COMMENT ON TABLE fast_calls IS 'Fast call contacts with display positions';
COMMENT ON COLUMN fast_calls.position IS 'Display position (order) for fast call';
