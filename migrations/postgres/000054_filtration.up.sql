-- Места фильтрации (filtration locations)
CREATE TABLE IF NOT EXISTS filtration_locations (
    id                 BIGSERIAL PRIMARY KEY,
    organization_id    BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    norm               DOUBLE PRECISION,
    sort_order         INT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id),
    updated_by_user_id BIGINT REFERENCES users(id),
    UNIQUE (organization_id, name)
);

CREATE INDEX IF NOT EXISTS idx_filtration_locations_org_id ON filtration_locations(organization_id);

CREATE TRIGGER set_timestamp_filtration_locations
    BEFORE UPDATE ON filtration_locations
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Пьезометры
CREATE TABLE IF NOT EXISTS piezometers (
    id                 BIGSERIAL PRIMARY KEY,
    organization_id    BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    type               TEXT NOT NULL CHECK (type IN ('pressure', 'non_pressure')),
    norm               DOUBLE PRECISION,
    sort_order         INT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id),
    updated_by_user_id BIGINT REFERENCES users(id),
    UNIQUE (organization_id, name)
);

CREATE INDEX IF NOT EXISTS idx_piezometers_org_id ON piezometers(organization_id);
CREATE INDEX IF NOT EXISTS idx_piezometers_org_type ON piezometers(organization_id, type);

CREATE TRIGGER set_timestamp_piezometers
    BEFORE UPDATE ON piezometers
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Замеры фильтрации
CREATE TABLE IF NOT EXISTS filtration_measurements (
    id                 BIGSERIAL PRIMARY KEY,
    location_id        BIGINT NOT NULL REFERENCES filtration_locations(id) ON DELETE CASCADE,
    date               DATE NOT NULL,
    flow_rate          DOUBLE PRECISION,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id),
    updated_by_user_id BIGINT REFERENCES users(id),
    UNIQUE (location_id, date)
);

CREATE INDEX IF NOT EXISTS idx_filtration_measurements_loc_date ON filtration_measurements(location_id, date);

CREATE TRIGGER set_timestamp_filtration_measurements
    BEFORE UPDATE ON filtration_measurements
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Замеры пьезометров
CREATE TABLE IF NOT EXISTS piezometer_measurements (
    id                 BIGSERIAL PRIMARY KEY,
    piezometer_id      BIGINT NOT NULL REFERENCES piezometers(id) ON DELETE CASCADE,
    date               DATE NOT NULL,
    level              DOUBLE PRECISION,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id),
    updated_by_user_id BIGINT REFERENCES users(id),
    UNIQUE (piezometer_id, date)
);

CREATE INDEX IF NOT EXISTS idx_piezometer_measurements_piezo_date ON piezometer_measurements(piezometer_id, date);

CREATE TRIGGER set_timestamp_piezometer_measurements
    BEFORE UPDATE ON piezometer_measurements
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
