CREATE TABLE reservoir_flood_config (
    id              BIGSERIAL PRIMARY KEY,
    organization_id BIGINT      NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,
    sort_order      INT         NOT NULL DEFAULT 0,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE reservoir_flood_hourly (
    id                   BIGSERIAL PRIMARY KEY,
    organization_id      BIGINT      NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    recorded_at          TIMESTAMPTZ NOT NULL,
    water_level_m        NUMERIC,
    water_volume_mln_m3  NUMERIC CHECK (water_volume_mln_m3 >= 0),
    inflow_m3s           NUMERIC CHECK (inflow_m3s >= 0),
    outflow_m3s          NUMERIC CHECK (outflow_m3s >= 0),
    ges_flow_m3s         NUMERIC CHECK (ges_flow_m3s >= 0),
    filtration_m3s       NUMERIC CHECK (filtration_m3s >= 0),
    idle_discharge_m3s   NUMERIC CHECK (idle_discharge_m3s >= 0),
    duty_name            TEXT,
    created_by_user_id   BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    updated_by_user_id   BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, recorded_at)
);

CREATE INDEX idx_reservoir_flood_hourly_org_time ON reservoir_flood_hourly (organization_id, recorded_at DESC);
CREATE INDEX idx_reservoir_flood_hourly_time     ON reservoir_flood_hourly (recorded_at DESC);
