-- Org Structure
CREATE TABLE org_units (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(20) NOT NULL DEFAULT 'department'
                    CHECK (type IN ('company','branch','division','department','section','group','team')),
    parent_id       BIGINT REFERENCES org_units(id) ON DELETE SET NULL,
    head_id         BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    department_id   BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    level           INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_org_units_parent_id ON org_units(parent_id);

CREATE TRIGGER set_timestamp_org_units
    BEFORE UPDATE ON org_units
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
