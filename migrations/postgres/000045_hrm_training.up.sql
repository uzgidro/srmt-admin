-- HRM Training Management

-- Training courses/programs
CREATE TABLE IF NOT EXISTS hrm_trainings (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,

    -- Training details
    training_type VARCHAR(50) NOT NULL, -- internal, external, online, workshop, certification
    category VARCHAR(100), -- technical, soft_skills, compliance, safety, leadership
    provider VARCHAR(255), -- Internal or external provider name
    instructor VARCHAR(255),

    -- Schedule
    start_date DATE,
    end_date DATE,
    duration_hours INTEGER,
    location VARCHAR(255), -- Physical location or online platform

    -- Capacity
    max_participants INTEGER,
    min_participants INTEGER,

    -- Cost
    cost_per_participant DECIMAL(15,2),
    currency VARCHAR(3) DEFAULT 'RUB',

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'planned', -- planned, registration, in_progress, completed, cancelled
    is_mandatory BOOLEAN DEFAULT FALSE,

    -- Materials
    materials_file_id BIGINT REFERENCES files(id) ON DELETE SET NULL,

    -- Responsible
    organizer_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_trainings_type ON hrm_trainings(training_type);
CREATE INDEX idx_hrm_trainings_category ON hrm_trainings(category);
CREATE INDEX idx_hrm_trainings_status ON hrm_trainings(status);
CREATE INDEX idx_hrm_trainings_dates ON hrm_trainings(start_date, end_date);
CREATE INDEX idx_hrm_trainings_mandatory ON hrm_trainings(is_mandatory) WHERE is_mandatory = TRUE;

CREATE TRIGGER set_timestamp_hrm_trainings
    BEFORE UPDATE ON hrm_trainings
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Training participants (enrollments)
CREATE TABLE IF NOT EXISTS hrm_training_participants (
    id BIGSERIAL PRIMARY KEY,
    training_id BIGINT NOT NULL REFERENCES hrm_trainings(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- Enrollment
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    enrolled_by BIGINT REFERENCES users(id) ON DELETE SET NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'enrolled', -- enrolled, attended, completed, no_show, cancelled

    -- Results
    attendance_percent INTEGER CHECK (attendance_percent IS NULL OR (attendance_percent >= 0 AND attendance_percent <= 100)),
    score DECIMAL(5,2),
    passed BOOLEAN,
    completed_at TIMESTAMPTZ,

    -- Feedback
    feedback_rating INTEGER CHECK (feedback_rating IS NULL OR (feedback_rating >= 1 AND feedback_rating <= 5)),
    feedback_text TEXT,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_training_participant UNIQUE (training_id, employee_id)
);

CREATE INDEX idx_hrm_training_participants_training ON hrm_training_participants(training_id);
CREATE INDEX idx_hrm_training_participants_employee ON hrm_training_participants(employee_id);
CREATE INDEX idx_hrm_training_participants_status ON hrm_training_participants(status);

CREATE TRIGGER set_timestamp_hrm_training_participants
    BEFORE UPDATE ON hrm_training_participants
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Employee certificates
CREATE TABLE IF NOT EXISTS hrm_certificates (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,
    training_id BIGINT REFERENCES hrm_trainings(id) ON DELETE SET NULL,

    -- Certificate info
    name VARCHAR(255) NOT NULL,
    issuer VARCHAR(255) NOT NULL,
    certificate_number VARCHAR(100),

    -- Dates
    issued_date DATE NOT NULL,
    expiry_date DATE,

    -- File
    file_id BIGINT REFERENCES files(id) ON DELETE SET NULL,

    -- Status
    is_verified BOOLEAN DEFAULT FALSE,
    verified_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    verified_at TIMESTAMPTZ,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_certificates_employee ON hrm_certificates(employee_id);
CREATE INDEX idx_hrm_certificates_training ON hrm_certificates(training_id);
CREATE INDEX idx_hrm_certificates_expiry ON hrm_certificates(expiry_date);
CREATE INDEX idx_hrm_certificates_verified ON hrm_certificates(is_verified);

CREATE TRIGGER set_timestamp_hrm_certificates
    BEFORE UPDATE ON hrm_certificates
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Individual Development Plans (IDP)
CREATE TABLE IF NOT EXISTS hrm_development_plans (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- Plan period
    title VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, active, completed, cancelled

    -- Review
    manager_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    overall_progress INTEGER DEFAULT 0 CHECK (overall_progress >= 0 AND overall_progress <= 100),
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_development_plans_employee ON hrm_development_plans(employee_id);
CREATE INDEX idx_hrm_development_plans_status ON hrm_development_plans(status);
CREATE INDEX idx_hrm_development_plans_dates ON hrm_development_plans(start_date, end_date);

CREATE TRIGGER set_timestamp_hrm_development_plans
    BEFORE UPDATE ON hrm_development_plans
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Development goals within IDP
CREATE TABLE IF NOT EXISTS hrm_development_goals (
    id BIGSERIAL PRIMARY KEY,
    plan_id BIGINT NOT NULL REFERENCES hrm_development_plans(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- Goal details
    title VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100), -- skill, certification, project, role

    -- Target
    target_date DATE,
    priority VARCHAR(20) DEFAULT 'normal', -- low, normal, high

    -- Progress
    status VARCHAR(20) NOT NULL DEFAULT 'not_started', -- not_started, in_progress, completed, cancelled
    progress INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    completed_at TIMESTAMPTZ,

    -- Related training/certification
    training_id BIGINT REFERENCES hrm_trainings(id) ON DELETE SET NULL,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_development_goals_plan ON hrm_development_goals(plan_id);
CREATE INDEX idx_hrm_development_goals_employee ON hrm_development_goals(employee_id);
CREATE INDEX idx_hrm_development_goals_status ON hrm_development_goals(status);

CREATE TRIGGER set_timestamp_hrm_development_goals
    BEFORE UPDATE ON hrm_development_goals
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_trainings IS 'Training courses and programs';
COMMENT ON TABLE hrm_training_participants IS 'Employee enrollment in trainings';
COMMENT ON TABLE hrm_certificates IS 'Employee certificates and certifications';
COMMENT ON TABLE hrm_development_plans IS 'Individual Development Plans (IDP)';
COMMENT ON TABLE hrm_development_goals IS 'Goals within development plans';
