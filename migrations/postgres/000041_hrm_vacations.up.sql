-- HRM Vacation Management

-- Vacation/leave types
CREATE TABLE IF NOT EXISTS hrm_vacation_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(20) NOT NULL UNIQUE,
    description TEXT,

    -- Days configuration
    default_days_per_year INTEGER DEFAULT 0,
    is_paid BOOLEAN DEFAULT TRUE,
    requires_approval BOOLEAN DEFAULT TRUE,
    can_carry_over BOOLEAN DEFAULT TRUE,
    max_carry_over_days INTEGER DEFAULT 0,

    is_active BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE TRIGGER set_timestamp_hrm_vacation_types
    BEFORE UPDATE ON hrm_vacation_types
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Insert default vacation types
INSERT INTO hrm_vacation_types (name, code, default_days_per_year, is_paid, can_carry_over, sort_order) VALUES
    ('Annual Leave', 'ANNUAL', 28, TRUE, TRUE, 1),
    ('Sick Leave', 'SICK', 0, TRUE, FALSE, 2),
    ('Unpaid Leave', 'UNPAID', 0, FALSE, FALSE, 3),
    ('Study Leave', 'STUDY', 14, TRUE, FALSE, 4),
    ('Maternity Leave', 'MATERNITY', 0, TRUE, FALSE, 5),
    ('Paternity Leave', 'PATERNITY', 14, TRUE, FALSE, 6),
    ('Compensatory Leave', 'COMP', 0, TRUE, FALSE, 7),
    ('Marriage Leave', 'MARRIAGE', 3, TRUE, FALSE, 8),
    ('Bereavement Leave', 'BEREAVEMENT', 3, TRUE, FALSE, 9);

-- Employee vacation balances per year
CREATE TABLE IF NOT EXISTS hrm_vacation_balances (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,
    vacation_type_id INTEGER NOT NULL REFERENCES hrm_vacation_types(id) ON DELETE RESTRICT,
    year INTEGER NOT NULL,

    -- Balance tracking
    entitled_days DECIMAL(5,1) NOT NULL DEFAULT 0, -- Days allowed for this year
    used_days DECIMAL(5,1) NOT NULL DEFAULT 0, -- Days already taken
    carried_over_days DECIMAL(5,1) NOT NULL DEFAULT 0, -- Days from previous year
    adjustment_days DECIMAL(5,1) NOT NULL DEFAULT 0, -- Manual adjustments

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_employee_vacation_year UNIQUE (employee_id, vacation_type_id, year)
);

CREATE INDEX idx_hrm_vacation_balances_employee ON hrm_vacation_balances(employee_id);
CREATE INDEX idx_hrm_vacation_balances_year ON hrm_vacation_balances(year);

CREATE TRIGGER set_timestamp_hrm_vacation_balances
    BEFORE UPDATE ON hrm_vacation_balances
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Vacation requests
CREATE TABLE IF NOT EXISTS hrm_vacations (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,
    vacation_type_id INTEGER NOT NULL REFERENCES hrm_vacation_types(id) ON DELETE RESTRICT,

    -- Period
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    days_count DECIMAL(5,1) NOT NULL, -- Actual working days

    -- Request status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected, cancelled
    reason TEXT, -- Reason for request (optional)
    rejection_reason TEXT, -- If rejected

    -- Approval workflow
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,

    -- Substitute employee during absence
    substitute_employee_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,

    -- Documentation
    supporting_document_id BIGINT REFERENCES files(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT check_vacation_dates CHECK (end_date >= start_date),
    CONSTRAINT check_vacation_days CHECK (days_count > 0)
);

CREATE INDEX idx_hrm_vacations_employee ON hrm_vacations(employee_id);
CREATE INDEX idx_hrm_vacations_type ON hrm_vacations(vacation_type_id);
CREATE INDEX idx_hrm_vacations_status ON hrm_vacations(status);
CREATE INDEX idx_hrm_vacations_dates ON hrm_vacations(start_date, end_date);
CREATE INDEX idx_hrm_vacations_approver ON hrm_vacations(approved_by);

CREATE TRIGGER set_timestamp_hrm_vacations
    BEFORE UPDATE ON hrm_vacations
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_vacation_types IS 'Types of leave/vacation available';
COMMENT ON TABLE hrm_vacation_balances IS 'Employee vacation balance per type per year';
COMMENT ON TABLE hrm_vacations IS 'Vacation/leave requests with approval workflow';
