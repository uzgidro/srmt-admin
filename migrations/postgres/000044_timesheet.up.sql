-- timesheet_entries: one record = one employee × one day
CREATE TABLE timesheet_entries (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    date            DATE NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'present'
                    CHECK (status IN ('present','absent','vacation','sick_leave',
                      'business_trip','remote','day_off','holiday','maternity',
                      'study_leave','unauthorized')),
    check_in        TIME,
    check_out       TIME,
    hours_worked    DECIMAL(5,2),
    overtime        DECIMAL(5,2) DEFAULT 0,
    is_weekend      BOOLEAN NOT NULL DEFAULT FALSE,
    is_holiday      BOOLEAN NOT NULL DEFAULT FALSE,
    note            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(employee_id, date)
);

-- holidays: reference of holiday days
CREATE TABLE holidays (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    date        DATE NOT NULL UNIQUE,
    type        VARCHAR(20) NOT NULL DEFAULT 'national'
                CHECK (type IN ('national','religious','company')),
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- timesheet_corrections: corrections with approve/reject workflow
CREATE TABLE timesheet_corrections (
    id                  BIGSERIAL PRIMARY KEY,
    employee_id         BIGINT NOT NULL REFERENCES contacts(id),
    date                DATE NOT NULL,
    original_status     VARCHAR(20),
    new_status          VARCHAR(20) NOT NULL,
    original_check_in   TIME,
    new_check_in        TIME,
    original_check_out  TIME,
    new_check_out       TIME,
    reason              TEXT NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','approved','rejected')),
    requested_by        BIGINT NOT NULL REFERENCES contacts(id),
    approved_by         BIGINT REFERENCES contacts(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_timesheet_entries_employee_date ON timesheet_entries(employee_id, date);
CREATE INDEX idx_timesheet_corrections_employee ON timesheet_corrections(employee_id);
CREATE INDEX idx_timesheet_corrections_status ON timesheet_corrections(status);

-- Triggers for updated_at
CREATE TRIGGER set_timesheet_entries_updated_at
    BEFORE UPDATE ON timesheet_entries
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timesheet_corrections_updated_at
    BEFORE UPDATE ON timesheet_corrections
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
