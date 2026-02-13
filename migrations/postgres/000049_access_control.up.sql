-- Access Control
CREATE TABLE access_zones (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    security_level  VARCHAR(20) NOT NULL DEFAULT 'low'
                    CHECK (security_level IN ('low','medium','high','restricted')),
    building        VARCHAR(255),
    floor           VARCHAR(50),
    max_occupancy   INTEGER NOT NULL DEFAULT 0,
    readers         JSONB NOT NULL DEFAULT '[]',
    schedules       JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE access_cards (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    card_number     VARCHAR(100) NOT NULL UNIQUE,
    status          VARCHAR(20) NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active','blocked','lost','expired','deactivated')),
    issued_date     DATE NOT NULL,
    expiry_date     DATE NOT NULL,
    access_zones    JSONB NOT NULL DEFAULT '[]',
    access_level    VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE access_logs (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    card_id         BIGINT REFERENCES access_cards(id) ON DELETE SET NULL,
    zone_id         BIGINT REFERENCES access_zones(id) ON DELETE SET NULL,
    reader_id       INTEGER,
    direction       VARCHAR(10) NOT NULL CHECK (direction IN ('entry','exit')),
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status          VARCHAR(20) NOT NULL DEFAULT 'granted'
                    CHECK (status IN ('granted','denied','error','forced')),
    denial_reason   TEXT
);

CREATE TABLE access_requests (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    zone_id         BIGINT REFERENCES access_zones(id) ON DELETE SET NULL,
    reason          TEXT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','approved','rejected')),
    approved_by     BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    rejection_reason TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_access_cards_employee_status ON access_cards(employee_id, status);
CREATE INDEX idx_access_logs_employee_timestamp ON access_logs(employee_id, timestamp);
CREATE INDEX idx_access_logs_zone_id ON access_logs(zone_id);
CREATE INDEX idx_access_requests_employee_status ON access_requests(employee_id, status);

CREATE TRIGGER set_timestamp_access_zones
    BEFORE UPDATE ON access_zones
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_access_cards
    BEFORE UPDATE ON access_cards
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_access_requests
    BEFORE UPDATE ON access_requests
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
