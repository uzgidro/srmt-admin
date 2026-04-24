CREATE TABLE ges_frozen_defaults (
    id              BIGSERIAL PRIMARY KEY,
    organization_id BIGINT      NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    field_name      TEXT        NOT NULL,
    frozen_value    NUMERIC     NOT NULL,
    frozen_by       BIGINT      REFERENCES users(id),
    frozen_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, field_name),
    CHECK (field_name IN (
        'daily_production_mln_kwh',
        'working_aggregates',
        'repair_aggregates',
        'modernization_aggregates',
        'water_level_m',
        'water_volume_mln_m3',
        'water_head_m',
        'reservoir_income_m3s',
        'total_outflow_m3s',
        'ges_flow_m3s'
    ))
);

CREATE INDEX idx_ges_frozen_defaults_org ON ges_frozen_defaults(organization_id);
