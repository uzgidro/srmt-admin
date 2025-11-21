-- Create Visits table
CREATE TABLE Visits (
                        id BIGSERIAL PRIMARY KEY,

    -- Organization reference
                        organization_id BIGINT NOT NULL REFERENCES Organizations(id) ON DELETE RESTRICT,

    -- Visit date and time
                        visit_date TIMESTAMPTZ NOT NULL,

    -- Description
                        description TEXT NOT NULL,

    -- Responsible person name
                        responsible_name VARCHAR(255) NOT NULL,

    -- Audit fields
                        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                        created_by_user_id BIGINT REFERENCES Users(id) ON DELETE SET NULL,
                        updated_at TIMESTAMPTZ
);

-- Trigger for auto-updating updated_at
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON Visits
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Indexes for fast queries
CREATE INDEX idx_visits_organization_id ON Visits(organization_id);
CREATE INDEX idx_visits_visit_date ON Visits(visit_date);
CREATE INDEX idx_visits_created_by_user_id ON Visits(created_by_user_id);
