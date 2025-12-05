CREATE TABLE reservoir_data
(
    id                   SERIAL PRIMARY KEY,
    organization_id      INTEGER     NOT NULL REFERENCES organizations (id) ON DELETE RESTRICT,
    income_m3_s          NUMERIC,
    release_m3_s         NUMERIC,
    level_m              NUMERIC,
    volume_mln_m3        NUMERIC,
    date                 TIMESTAMPTZ NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id   INTEGER     NOT NULL REFERENCES users (id),
    updated_by_user_id   INTEGER REFERENCES users (id)
);

CREATE INDEX idx_reservoir_data_date ON reservoir_data (date);
CREATE INDEX idx_reservoir_data_org_id ON reservoir_data (organization_id);
