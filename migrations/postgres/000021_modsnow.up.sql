CREATE TABLE modsnow
(
    id                   SERIAL PRIMARY KEY,
    organization_id      INTEGER     NOT NULL REFERENCES organizations (id) ON DELETE RESTRICT,
    date                 DATE        NOT NULL,
    cover                NUMERIC,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id   INTEGER     NOT NULL REFERENCES users (id),
    updated_by_user_id   INTEGER REFERENCES users (id),
    UNIQUE (organization_id, date)
);

CREATE INDEX idx_modsnow_date ON modsnow (date);
CREATE INDEX idx_modsnow_org_id ON modsnow (organization_id);
