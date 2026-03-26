-- Manual comparison tables: current + historical values side by side,
-- historical date is a free-text label (not FK to real measurements).

CREATE TABLE manual_comparison_filter (
    id                   BIGSERIAL PRIMARY KEY,
    organization_id      BIGINT NOT NULL REFERENCES organizations(id),
    location_id          BIGINT NOT NULL REFERENCES filtration_locations(id),
    date                 DATE NOT NULL,
    flow_rate            DOUBLE PRECISION,
    historical_flow_rate DOUBLE PRECISION,
    created_by_user_id   BIGINT NOT NULL,
    updated_by_user_id   BIGINT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (location_id, date)
);

CREATE TABLE manual_comparison_piezo (
    id                   BIGSERIAL PRIMARY KEY,
    organization_id      BIGINT NOT NULL REFERENCES organizations(id),
    piezometer_id        BIGINT NOT NULL REFERENCES piezometers(id),
    date                 DATE NOT NULL,
    level                DOUBLE PRECISION,
    anomaly              BOOLEAN NOT NULL DEFAULT FALSE,
    historical_level     DOUBLE PRECISION,
    created_by_user_id   BIGINT NOT NULL,
    updated_by_user_id   BIGINT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (piezometer_id, date)
);

CREATE TABLE manual_comparison_dates (
    id                     BIGSERIAL PRIMARY KEY,
    organization_id        BIGINT NOT NULL REFERENCES organizations(id),
    date                   DATE NOT NULL,
    historical_filter_date TEXT NOT NULL DEFAULT '',
    historical_piezo_date  TEXT NOT NULL DEFAULT '',
    created_by_user_id     BIGINT NOT NULL,
    updated_by_user_id     BIGINT NOT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, date)
);

CREATE INDEX idx_mcf_org_date ON manual_comparison_filter(organization_id, date);
CREATE INDEX idx_mcp_org_date ON manual_comparison_piezo(organization_id, date);
