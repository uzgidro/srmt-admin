-- HRM Employee Extension
-- Links to existing contacts and users tables

CREATE TABLE IF NOT EXISTS hrm_employees (
    id BIGSERIAL PRIMARY KEY,
    contact_id BIGINT NOT NULL REFERENCES contacts(id) ON DELETE RESTRICT,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

    -- Employment info
    employee_number VARCHAR(50) UNIQUE,
    hire_date DATE NOT NULL,
    termination_date DATE,
    employment_type VARCHAR(50) NOT NULL DEFAULT 'full_time', -- full_time, part_time, contract, intern
    employment_status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, on_leave, suspended, terminated

    -- Work schedule
    work_schedule VARCHAR(50) DEFAULT '5/2', -- 5/2, 2/2, shift, flexible
    work_hours_per_week DECIMAL(4,1) DEFAULT 40,

    -- Manager hierarchy
    manager_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,

    -- Probation
    probation_end_date DATE,
    probation_passed BOOLEAN DEFAULT FALSE,

    -- Additional info
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_contact_employee UNIQUE (contact_id)
);

-- Index for common queries
CREATE INDEX idx_hrm_employees_contact_id ON hrm_employees(contact_id);
CREATE INDEX idx_hrm_employees_user_id ON hrm_employees(user_id);
CREATE INDEX idx_hrm_employees_manager_id ON hrm_employees(manager_id);
CREATE INDEX idx_hrm_employees_status ON hrm_employees(employment_status);
CREATE INDEX idx_hrm_employees_hire_date ON hrm_employees(hire_date);

-- Trigger for updated_at
CREATE TRIGGER set_timestamp_hrm_employees
    BEFORE UPDATE ON hrm_employees
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_employees IS 'Extended employee information linked to contacts';
COMMENT ON COLUMN hrm_employees.contact_id IS 'Link to contacts table for personal info';
COMMENT ON COLUMN hrm_employees.user_id IS 'Link to users table for system access';
COMMENT ON COLUMN hrm_employees.employee_number IS 'Unique employee identifier/badge number';
