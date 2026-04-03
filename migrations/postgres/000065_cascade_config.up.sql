CREATE TABLE cascade_config (
    id              BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL UNIQUE
                    REFERENCES organizations(id) ON DELETE RESTRICT,
    latitude        DOUBLE PRECISION,
    longitude       DOUBLE PRECISION,
    sort_order      INT DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
