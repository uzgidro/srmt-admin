-- Create investment_status table with predefined statuses
CREATE TABLE IF NOT EXISTS investment_status (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT
);

-- Insert predefined investment statuses
INSERT INTO investment_status (name, description) VALUES
    ('Planned', 'Investment is planned but not yet started'),
    ('In Progress', 'Investment activities are currently underway'),
    ('Completed', 'Investment has been successfully completed'),
    ('Cancelled', 'Investment was cancelled or abandoned')
ON CONFLICT (name) DO NOTHING;

-- Create investments table
CREATE TABLE IF NOT EXISTS investments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status_id INTEGER NOT NULL REFERENCES investment_status(id) ON DELETE RESTRICT,
    cost NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    comments TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ
);

-- Create trigger for updated_at (trigger function already exists from migration 000010)
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON investments
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Create junction table for many-to-many file relationship
CREATE TABLE IF NOT EXISTS investment_file_links (
    investment_id BIGINT NOT NULL REFERENCES investments(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (investment_id, file_id)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_investments_status ON investments(status_id);
CREATE INDEX IF NOT EXISTS idx_investments_name ON investments(name);
CREATE INDEX IF NOT EXISTS idx_investments_created_at ON investments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_investments_created_by ON investments(created_by_user_id);
CREATE INDEX IF NOT EXISTS idx_investment_file_links_investment ON investment_file_links(investment_id);
CREATE INDEX IF NOT EXISTS idx_investment_file_links_file ON investment_file_links(file_id);
