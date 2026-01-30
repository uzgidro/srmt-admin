-- HRM Salary Management

-- Salary structure (base configuration per employee)
CREATE TABLE IF NOT EXISTS hrm_salary_structures (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- Base salary
    base_salary DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    pay_frequency VARCHAR(20) NOT NULL DEFAULT 'monthly', -- monthly, bi_weekly, weekly

    -- Allowances (JSON for flexibility)
    allowances JSONB DEFAULT '[]', -- [{type: "transport", amount: 5000}, {type: "food", amount: 3000}]

    effective_from DATE NOT NULL,
    effective_to DATE, -- NULL means current

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_salary_structures_employee ON hrm_salary_structures(employee_id);
CREATE INDEX idx_hrm_salary_structures_effective ON hrm_salary_structures(effective_from, effective_to);

CREATE TRIGGER set_timestamp_hrm_salary_structures
    BEFORE UPDATE ON hrm_salary_structures
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Monthly salary records (calculated payroll)
CREATE TABLE IF NOT EXISTS hrm_salaries (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- Period
    year INTEGER NOT NULL,
    month INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),

    -- Amounts
    base_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    allowances_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    bonuses_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    deductions_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    gross_amount DECIMAL(15,2) NOT NULL DEFAULT 0, -- base + allowances + bonuses
    tax_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    net_amount DECIMAL(15,2) NOT NULL DEFAULT 0, -- gross - deductions - tax

    -- Work time
    worked_days INTEGER NOT NULL DEFAULT 0,
    total_work_days INTEGER NOT NULL DEFAULT 0,
    overtime_hours DECIMAL(5,1) DEFAULT 0,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, calculated, approved, paid
    calculated_at TIMESTAMPTZ,
    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_employee_salary_period UNIQUE (employee_id, year, month)
);

CREATE INDEX idx_hrm_salaries_employee ON hrm_salaries(employee_id);
CREATE INDEX idx_hrm_salaries_period ON hrm_salaries(year, month);
CREATE INDEX idx_hrm_salaries_status ON hrm_salaries(status);

CREATE TRIGGER set_timestamp_hrm_salaries
    BEFORE UPDATE ON hrm_salaries
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Bonus entries
CREATE TABLE IF NOT EXISTS hrm_salary_bonuses (
    id BIGSERIAL PRIMARY KEY,
    salary_id BIGINT REFERENCES hrm_salaries(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    bonus_type VARCHAR(100) NOT NULL, -- performance, project, annual, referral, etc.
    amount DECIMAL(15,2) NOT NULL,
    description TEXT,

    -- Optional period (if not linked to specific salary record)
    year INTEGER,
    month INTEGER CHECK (month IS NULL OR (month >= 1 AND month <= 12)),

    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_salary_bonuses_salary ON hrm_salary_bonuses(salary_id);
CREATE INDEX idx_hrm_salary_bonuses_employee ON hrm_salary_bonuses(employee_id);
CREATE INDEX idx_hrm_salary_bonuses_period ON hrm_salary_bonuses(year, month);

CREATE TRIGGER set_timestamp_hrm_salary_bonuses
    BEFORE UPDATE ON hrm_salary_bonuses
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Deduction entries
CREATE TABLE IF NOT EXISTS hrm_salary_deductions (
    id BIGSERIAL PRIMARY KEY,
    salary_id BIGINT REFERENCES hrm_salaries(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    deduction_type VARCHAR(100) NOT NULL, -- tax, insurance, loan, penalty, absence, etc.
    amount DECIMAL(15,2) NOT NULL,
    description TEXT,

    -- Optional period (if not linked to specific salary record)
    year INTEGER,
    month INTEGER CHECK (month IS NULL OR (month >= 1 AND month <= 12)),

    -- For recurring deductions
    is_recurring BOOLEAN DEFAULT FALSE,
    recurring_until DATE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_salary_deductions_salary ON hrm_salary_deductions(salary_id);
CREATE INDEX idx_hrm_salary_deductions_employee ON hrm_salary_deductions(employee_id);
CREATE INDEX idx_hrm_salary_deductions_period ON hrm_salary_deductions(year, month);
CREATE INDEX idx_hrm_salary_deductions_recurring ON hrm_salary_deductions(is_recurring) WHERE is_recurring = TRUE;

CREATE TRIGGER set_timestamp_hrm_salary_deductions
    BEFORE UPDATE ON hrm_salary_deductions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_salary_structures IS 'Employee salary configuration (base + allowances)';
COMMENT ON TABLE hrm_salaries IS 'Monthly calculated salary records';
COMMENT ON TABLE hrm_salary_bonuses IS 'Bonus payments to employees';
COMMENT ON TABLE hrm_salary_deductions IS 'Deductions from employee salary';
