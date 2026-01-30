-- HRM Timesheet Management

-- Public holidays
CREATE TABLE IF NOT EXISTS hrm_holidays (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    year INTEGER NOT NULL,
    is_working_day BOOLEAN DEFAULT FALSE, -- For special working Saturdays

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_holiday_date UNIQUE (date)
);

CREATE INDEX idx_hrm_holidays_year ON hrm_holidays(year);
CREATE INDEX idx_hrm_holidays_date ON hrm_holidays(date);

CREATE TRIGGER set_timestamp_hrm_holidays
    BEFORE UPDATE ON hrm_holidays
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Monthly timesheet summary
CREATE TABLE IF NOT EXISTS hrm_timesheets (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),

    -- Summary hours
    total_work_days INTEGER NOT NULL DEFAULT 0,
    total_worked_days INTEGER NOT NULL DEFAULT 0,
    total_hours DECIMAL(6,1) NOT NULL DEFAULT 0,
    overtime_hours DECIMAL(6,1) NOT NULL DEFAULT 0,
    sick_days INTEGER NOT NULL DEFAULT 0,
    vacation_days INTEGER NOT NULL DEFAULT 0,
    absence_days INTEGER NOT NULL DEFAULT 0,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, submitted, approved, rejected
    submitted_at TIMESTAMPTZ,
    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    rejection_reason TEXT,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_employee_timesheet_period UNIQUE (employee_id, year, month)
);

CREATE INDEX idx_hrm_timesheets_employee ON hrm_timesheets(employee_id);
CREATE INDEX idx_hrm_timesheets_period ON hrm_timesheets(year, month);
CREATE INDEX idx_hrm_timesheets_status ON hrm_timesheets(status);

CREATE TRIGGER set_timestamp_hrm_timesheets
    BEFORE UPDATE ON hrm_timesheets
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Daily timesheet entries
CREATE TABLE IF NOT EXISTS hrm_timesheet_entries (
    id BIGSERIAL PRIMARY KEY,
    timesheet_id BIGINT NOT NULL REFERENCES hrm_timesheets(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,
    date DATE NOT NULL,

    -- Time tracking
    check_in TIME,
    check_out TIME,
    break_minutes INTEGER DEFAULT 0,
    worked_hours DECIMAL(4,1) DEFAULT 0,
    overtime_hours DECIMAL(4,1) DEFAULT 0,

    -- Day type
    day_type VARCHAR(20) NOT NULL DEFAULT 'work', -- work, weekend, holiday, vacation, sick, absence, remote
    is_remote BOOLEAN DEFAULT FALSE,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_employee_entry_date UNIQUE (employee_id, date)
);

CREATE INDEX idx_hrm_timesheet_entries_timesheet ON hrm_timesheet_entries(timesheet_id);
CREATE INDEX idx_hrm_timesheet_entries_employee ON hrm_timesheet_entries(employee_id);
CREATE INDEX idx_hrm_timesheet_entries_date ON hrm_timesheet_entries(date);
CREATE INDEX idx_hrm_timesheet_entries_day_type ON hrm_timesheet_entries(day_type);

CREATE TRIGGER set_timestamp_hrm_timesheet_entries
    BEFORE UPDATE ON hrm_timesheet_entries
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Timesheet correction requests
CREATE TABLE IF NOT EXISTS hrm_timesheet_corrections (
    id BIGSERIAL PRIMARY KEY,
    entry_id BIGINT NOT NULL REFERENCES hrm_timesheet_entries(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- Original values
    original_check_in TIME,
    original_check_out TIME,
    original_day_type VARCHAR(20),

    -- Requested values
    requested_check_in TIME,
    requested_check_out TIME,
    requested_day_type VARCHAR(20),

    reason TEXT NOT NULL,

    -- Approval
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    rejection_reason TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_timesheet_corrections_entry ON hrm_timesheet_corrections(entry_id);
CREATE INDEX idx_hrm_timesheet_corrections_employee ON hrm_timesheet_corrections(employee_id);
CREATE INDEX idx_hrm_timesheet_corrections_status ON hrm_timesheet_corrections(status);

CREATE TRIGGER set_timestamp_hrm_timesheet_corrections
    BEFORE UPDATE ON hrm_timesheet_corrections
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_holidays IS 'Public holidays and special working days';
COMMENT ON TABLE hrm_timesheets IS 'Monthly timesheet summary per employee';
COMMENT ON TABLE hrm_timesheet_entries IS 'Daily work time entries';
COMMENT ON TABLE hrm_timesheet_corrections IS 'Requests to correct timesheet entries';
