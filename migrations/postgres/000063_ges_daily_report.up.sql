-- GES static configuration (one row per GES organization)
CREATE TABLE ges_config (
    id                    BIGSERIAL PRIMARY KEY,
    organization_id       BIGINT    NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    installed_capacity_mwt NUMERIC  NOT NULL DEFAULT 0,
    total_aggregates      INT       NOT NULL DEFAULT 0,
    has_reservoir         BOOLEAN   NOT NULL DEFAULT FALSE,
    sort_order            INT       NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id)
);

-- Daily operational data (one row per GES per date)
CREATE TABLE ges_daily_data (
    id                       BIGSERIAL PRIMARY KEY,
    organization_id          BIGINT    NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    date                     DATE      NOT NULL,
    daily_production_mln_kwh NUMERIC   NOT NULL DEFAULT 0,
    working_aggregates       INT       NOT NULL DEFAULT 0,
    water_level_m            NUMERIC,
    water_volume_mln_m3      NUMERIC,
    water_head_m             NUMERIC,
    reservoir_income_m3s     NUMERIC,
    total_outflow_m3s        NUMERIC,
    ges_flow_m3s             NUMERIC,
    temperature              NUMERIC,
    weather_condition        TEXT,
    created_by_user_id       BIGINT    NOT NULL REFERENCES users(id),
    updated_by_user_id       BIGINT    NOT NULL REFERENCES users(id),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, date)
);

CREATE INDEX idx_ges_daily_data_date ON ges_daily_data(date);
CREATE INDEX idx_ges_daily_data_org_date ON ges_daily_data(organization_id, date);

-- Production plans (one row per GES per month)
CREATE TABLE ges_production_plan (
    id                    BIGSERIAL PRIMARY KEY,
    organization_id       BIGINT    NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    year                  INT       NOT NULL,
    month                 INT       NOT NULL CHECK (month >= 1 AND month <= 12),
    plan_mln_kwh          NUMERIC   NOT NULL DEFAULT 0,
    created_by_user_id    BIGINT    NOT NULL REFERENCES users(id),
    updated_by_user_id    BIGINT    NOT NULL REFERENCES users(id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, year, month)
);

CREATE INDEX idx_ges_production_plan_year ON ges_production_plan(year);
