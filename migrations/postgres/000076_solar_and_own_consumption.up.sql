-- Block B: own_consumption column for all stations (ges/mini/micro share ges_daily_data).
ALTER TABLE ges_daily_data
    ADD COLUMN own_consumption_kwh NUMERIC CHECK (own_consumption_kwh >= 0);

-- Block A: solar config (per-station).
CREATE TABLE solar_config (
    id                    BIGSERIAL PRIMARY KEY,
    organization_id       BIGINT      NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE RESTRICT,
    installed_capacity_kw NUMERIC     NOT NULL DEFAULT 0 CHECK (installed_capacity_kw >= 0),
    sort_order            INT         NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Block A: solar daily generation.
CREATE TABLE solar_daily_data (
    id                  BIGSERIAL PRIMARY KEY,
    organization_id     BIGINT      NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    date                DATE        NOT NULL,
    generation_kwh      NUMERIC     CHECK (generation_kwh >= 0),
    grid_export_kwh     NUMERIC     CHECK (grid_export_kwh >= 0),
    created_by_user_id  BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    updated_by_user_id  BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, date)
);
CREATE INDEX idx_solar_daily_org_date ON solar_daily_data (organization_id, date DESC);
CREATE INDEX idx_solar_daily_date     ON solar_daily_data (date DESC);

-- Block A: solar monthly plan in thousands of kWh.
CREATE TABLE solar_production_plan (
    id                  BIGSERIAL PRIMARY KEY,
    organization_id     BIGINT      NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    year                INT         NOT NULL,
    month               INT         NOT NULL CHECK (month >= 1 AND month <= 12),
    plan_thousand_kwh   NUMERIC     NOT NULL DEFAULT 0 CHECK (plan_thousand_kwh >= 0),
    created_by_user_id  BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    updated_by_user_id  BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, year, month)
);
