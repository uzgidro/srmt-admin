-- HRM Access Control (Physical Access)

-- Access zones (areas in buildings)
CREATE TABLE IF NOT EXISTS hrm_access_zones (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(30) UNIQUE,
    description TEXT,

    -- Location
    building VARCHAR(100),
    floor VARCHAR(20),

    -- Security level
    security_level INTEGER DEFAULT 1 CHECK (security_level >= 1 AND security_level <= 5),

    -- Time restrictions (JSON for flexibility)
    access_schedule JSONB, -- {mon: {start: "08:00", end: "20:00"}, ...}

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_access_zones_active ON hrm_access_zones(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_hrm_access_zones_security ON hrm_access_zones(security_level);

CREATE TRIGGER set_timestamp_hrm_access_zones
    BEFORE UPDATE ON hrm_access_zones
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Insert default zones
INSERT INTO hrm_access_zones (name, code, security_level) VALUES
    ('Main Entrance', 'MAIN', 1),
    ('Office Area', 'OFFICE', 2),
    ('Server Room', 'SERVER', 5),
    ('Meeting Rooms', 'MEETING', 2),
    ('Executive Floor', 'EXECUTIVE', 4),
    ('Parking', 'PARKING', 1);

-- Employee access cards
CREATE TABLE IF NOT EXISTS hrm_access_cards (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- Card info
    card_number VARCHAR(100) NOT NULL UNIQUE,
    card_type VARCHAR(50) DEFAULT 'standard', -- standard, temporary, visitor, contractor

    -- Validity
    issued_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expiry_date DATE,
    is_active BOOLEAN DEFAULT TRUE,

    -- Deactivation
    deactivated_at TIMESTAMPTZ,
    deactivation_reason TEXT,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_access_cards_employee ON hrm_access_cards(employee_id);
CREATE INDEX idx_hrm_access_cards_number ON hrm_access_cards(card_number);
CREATE INDEX idx_hrm_access_cards_active ON hrm_access_cards(is_active) WHERE is_active = TRUE;

CREATE TRIGGER set_timestamp_hrm_access_cards
    BEFORE UPDATE ON hrm_access_cards
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Card-zone access permissions
CREATE TABLE IF NOT EXISTS hrm_card_zone_access (
    id BIGSERIAL PRIMARY KEY,
    card_id BIGINT NOT NULL REFERENCES hrm_access_cards(id) ON DELETE CASCADE,
    zone_id INTEGER NOT NULL REFERENCES hrm_access_zones(id) ON DELETE CASCADE,

    -- Custom schedule (overrides zone default)
    custom_schedule JSONB,

    -- Validity for this zone
    valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
    valid_until DATE,

    granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_card_zone UNIQUE (card_id, zone_id)
);

CREATE INDEX idx_hrm_card_zone_access_card ON hrm_card_zone_access(card_id);
CREATE INDEX idx_hrm_card_zone_access_zone ON hrm_card_zone_access(zone_id);

CREATE TRIGGER set_timestamp_hrm_card_zone_access
    BEFORE UPDATE ON hrm_card_zone_access
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Access logs
CREATE TABLE IF NOT EXISTS hrm_access_logs (
    id BIGSERIAL PRIMARY KEY,
    card_id BIGINT REFERENCES hrm_access_cards(id) ON DELETE SET NULL,
    employee_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,
    zone_id INTEGER NOT NULL REFERENCES hrm_access_zones(id) ON DELETE RESTRICT,

    -- Event
    event_type VARCHAR(20) NOT NULL, -- entry, exit, denied
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Device info
    device_id VARCHAR(100),
    device_name VARCHAR(100),

    -- For denied attempts
    denial_reason TEXT,

    -- Additional data
    card_number VARCHAR(100), -- Denormalized for audit
    metadata JSONB
);

-- Partition by month for performance
CREATE INDEX idx_hrm_access_logs_card ON hrm_access_logs(card_id);
CREATE INDEX idx_hrm_access_logs_employee ON hrm_access_logs(employee_id);
CREATE INDEX idx_hrm_access_logs_zone ON hrm_access_logs(zone_id);
CREATE INDEX idx_hrm_access_logs_time ON hrm_access_logs(event_time);
CREATE INDEX idx_hrm_access_logs_event ON hrm_access_logs(event_type);

COMMENT ON TABLE hrm_access_zones IS 'Physical access zones/areas';
COMMENT ON TABLE hrm_access_cards IS 'Employee access cards';
COMMENT ON TABLE hrm_card_zone_access IS 'Card permissions for zones';
COMMENT ON TABLE hrm_access_logs IS 'Access event logs (entry/exit/denied)';
