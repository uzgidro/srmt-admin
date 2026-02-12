-- HRM Module MVP: Dashboard, Personnel Records, Vacation Management
-- 7 tables + 3 roles

BEGIN;

-- ============================================================
-- 1. personnel_records
-- ============================================================
CREATE TABLE IF NOT EXISTS personnel_records (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    tab_number      VARCHAR(50) NOT NULL,
    hire_date       DATE NOT NULL,
    department_id   BIGINT NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
    position_id     BIGINT NOT NULL REFERENCES positions(id) ON DELETE RESTRICT,
    contract_type   VARCHAR(20) NOT NULL CHECK (contract_type IN ('permanent', 'temporary', 'contract')),
    contract_end_date DATE,
    status          VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'on_leave', 'dismissed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_personnel_records_employee UNIQUE (employee_id)
);

CREATE INDEX idx_personnel_records_department ON personnel_records(department_id);
CREATE INDEX idx_personnel_records_position ON personnel_records(position_id);
CREATE INDEX idx_personnel_records_status ON personnel_records(status);

CREATE TRIGGER set_timestamp_personnel_records
    BEFORE UPDATE ON personnel_records
    FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- ============================================================
-- 2. personnel_documents
-- ============================================================
CREATE TABLE IF NOT EXISTS personnel_documents (
    id          BIGSERIAL PRIMARY KEY,
    record_id   BIGINT NOT NULL REFERENCES personnel_records(id) ON DELETE CASCADE,
    type        VARCHAR(50) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    file_url    TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_personnel_documents_record ON personnel_documents(record_id);

-- ============================================================
-- 3. personnel_transfers
-- ============================================================
CREATE TABLE IF NOT EXISTS personnel_transfers (
    id                  BIGSERIAL PRIMARY KEY,
    record_id           BIGINT NOT NULL REFERENCES personnel_records(id) ON DELETE CASCADE,
    from_department_id  BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    to_department_id    BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    from_position_id    BIGINT REFERENCES positions(id) ON DELETE SET NULL,
    to_position_id      BIGINT REFERENCES positions(id) ON DELETE SET NULL,
    transfer_date       DATE NOT NULL,
    order_number        VARCHAR(100),
    reason              TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_personnel_transfers_record ON personnel_transfers(record_id);
CREATE INDEX idx_personnel_transfers_date ON personnel_transfers(transfer_date);

-- ============================================================
-- 4. vacations
-- ============================================================
CREATE TABLE IF NOT EXISTS vacations (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    vacation_type   VARCHAR(20) NOT NULL CHECK (vacation_type IN ('annual', 'additional', 'study', 'unpaid', 'maternity', 'comp')),
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    days            INTEGER NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'approved', 'rejected', 'cancelled', 'active', 'completed')),
    reason          TEXT,
    rejection_reason TEXT,
    approved_by     BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    approved_at     TIMESTAMPTZ,
    substitute_id   BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    created_by      BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_vacations_dates CHECK (end_date >= start_date)
);

CREATE INDEX idx_vacations_employee ON vacations(employee_id);
CREATE INDEX idx_vacations_status ON vacations(status);
CREATE INDEX idx_vacations_dates ON vacations(start_date, end_date);
CREATE INDEX idx_vacations_approved_by ON vacations(approved_by);
CREATE INDEX idx_vacations_substitute ON vacations(substitute_id);

CREATE TRIGGER set_timestamp_vacations
    BEFORE UPDATE ON vacations
    FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- ============================================================
-- 5. vacation_balances
-- ============================================================
CREATE TABLE IF NOT EXISTS vacation_balances (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    year            INTEGER NOT NULL,
    total_days      INTEGER NOT NULL DEFAULT 0,
    used_days       INTEGER NOT NULL DEFAULT 0,
    pending_days    INTEGER NOT NULL DEFAULT 0,
    remaining_days  INTEGER NOT NULL DEFAULT 0,
    carried_over    INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_vacation_balances_employee_year UNIQUE (employee_id, year)
);

CREATE INDEX idx_vacation_balances_employee ON vacation_balances(employee_id);
CREATE INDEX idx_vacation_balances_year ON vacation_balances(year);

CREATE TRIGGER set_timestamp_vacation_balances
    BEFORE UPDATE ON vacation_balances
    FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- ============================================================
-- 6. hrm_notifications
-- ============================================================
CREATE TABLE IF NOT EXISTS hrm_notifications (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    title       VARCHAR(255) NOT NULL,
    message     TEXT NOT NULL,
    type        VARCHAR(20) NOT NULL DEFAULT 'info' CHECK (type IN ('info', 'warning', 'success', 'error', 'task')),
    read        BOOLEAN NOT NULL DEFAULT FALSE,
    read_at     TIMESTAMPTZ,
    link        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_hrm_notifications_user ON hrm_notifications(user_id);
CREATE INDEX idx_hrm_notifications_read ON hrm_notifications(user_id, read);

-- ============================================================
-- 7. department_blocked_periods
-- ============================================================
CREATE TABLE IF NOT EXISTS department_blocked_periods (
    id              BIGSERIAL PRIMARY KEY,
    department_id   BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    reason          TEXT,
    created_by      BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_blocked_periods_dates CHECK (end_date >= start_date)
);

CREATE INDEX idx_blocked_periods_department ON department_blocked_periods(department_id);
CREATE INDEX idx_blocked_periods_dates ON department_blocked_periods(start_date, end_date);

-- ============================================================
-- Seed HRM roles
-- ============================================================
INSERT INTO roles (name) VALUES
    ('hrm_admin'),
    ('hrm_manager'),
    ('hrm_employee')
ON CONFLICT DO NOTHING;

COMMIT;
