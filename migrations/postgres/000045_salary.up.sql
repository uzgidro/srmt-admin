-- salary_structures: employee salary structure (base + allowances)
CREATE TABLE salary_structures (
    id                      BIGSERIAL PRIMARY KEY,
    employee_id             BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    base_salary             DECIMAL(12,2) NOT NULL DEFAULT 0,
    regional_allowance      DECIMAL(12,2) NOT NULL DEFAULT 0,
    seniority_allowance     DECIMAL(12,2) NOT NULL DEFAULT 0,
    qualification_allowance DECIMAL(12,2) NOT NULL DEFAULT 0,
    hazard_allowance        DECIMAL(12,2) NOT NULL DEFAULT 0,
    night_shift_allowance   DECIMAL(12,2) NOT NULL DEFAULT 0,
    effective_from          DATE NOT NULL,
    effective_to            DATE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_salary_structures_employee_period UNIQUE (employee_id, effective_from)
);

-- salaries: monthly payroll record
CREATE TABLE salaries (
    id                      BIGSERIAL PRIMARY KEY,
    employee_id             BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    period_month            INTEGER NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    period_year             INTEGER NOT NULL CHECK (period_year >= 2020),
    base_salary             DECIMAL(12,2) NOT NULL DEFAULT 0,
    regional_allowance      DECIMAL(12,2) NOT NULL DEFAULT 0,
    seniority_allowance     DECIMAL(12,2) NOT NULL DEFAULT 0,
    qualification_allowance DECIMAL(12,2) NOT NULL DEFAULT 0,
    hazard_allowance        DECIMAL(12,2) NOT NULL DEFAULT 0,
    night_shift_allowance   DECIMAL(12,2) NOT NULL DEFAULT 0,
    overtime_amount         DECIMAL(12,2) NOT NULL DEFAULT 0,
    bonus_amount            DECIMAL(12,2) NOT NULL DEFAULT 0,
    gross_salary            DECIMAL(12,2) NOT NULL DEFAULT 0,
    ndfl                    DECIMAL(12,2) NOT NULL DEFAULT 0,
    social_tax              DECIMAL(12,2) NOT NULL DEFAULT 0,
    pension_fund            DECIMAL(12,2) NOT NULL DEFAULT 0,
    health_insurance        DECIMAL(12,2) NOT NULL DEFAULT 0,
    trade_union             DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_deductions        DECIMAL(12,2) NOT NULL DEFAULT 0,
    net_salary              DECIMAL(12,2) NOT NULL DEFAULT 0,
    work_days               INTEGER NOT NULL DEFAULT 0,
    actual_days             INTEGER NOT NULL DEFAULT 0,
    overtime_hours          DECIMAL(5,2) NOT NULL DEFAULT 0,
    status                  VARCHAR(20) NOT NULL DEFAULT 'draft'
                            CHECK (status IN ('draft','calculated','approved','paid','cancelled')),
    calculated_at           TIMESTAMPTZ,
    approved_by             BIGINT REFERENCES contacts(id),
    approved_at             TIMESTAMPTZ,
    paid_at                 TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_salaries_employee_period UNIQUE (employee_id, period_year, period_month)
);

-- salary_bonuses: bonuses linked to a salary record
CREATE TABLE salary_bonuses (
    id          BIGSERIAL PRIMARY KEY,
    salary_id   BIGINT NOT NULL REFERENCES salaries(id) ON DELETE CASCADE,
    bonus_type  VARCHAR(20) NOT NULL
                CHECK (bonus_type IN ('performance','quarterly','annual','holiday','project','overtime','other')),
    amount      DECIMAL(12,2) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- salary_deductions: deductions linked to a salary record
CREATE TABLE salary_deductions (
    id              BIGSERIAL PRIMARY KEY,
    salary_id       BIGINT NOT NULL REFERENCES salaries(id) ON DELETE CASCADE,
    deduction_type  VARCHAR(20) NOT NULL
                    CHECK (deduction_type IN ('tax','pension','insurance','loan','alimony','fine','advance','other')),
    amount          DECIMAL(12,2) NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_salaries_employee_id ON salaries(employee_id);
CREATE INDEX idx_salaries_period ON salaries(period_year, period_month);
CREATE INDEX idx_salaries_status ON salaries(status);
CREATE INDEX idx_salary_structures_employee_id ON salary_structures(employee_id);
CREATE INDEX idx_salary_bonuses_salary_id ON salary_bonuses(salary_id);
CREATE INDEX idx_salary_deductions_salary_id ON salary_deductions(salary_id);

-- Triggers for updated_at
CREATE TRIGGER set_timestamp_salaries
    BEFORE UPDATE ON salaries
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_salary_structures
    BEFORE UPDATE ON salary_structures
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
