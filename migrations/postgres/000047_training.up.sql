-- 1. trainings
CREATE TABLE trainings (
    id                  BIGSERIAL PRIMARY KEY,
    title               VARCHAR(255) NOT NULL,
    description         TEXT,
    type                VARCHAR(20) NOT NULL DEFAULT 'internal'
                        CHECK (type IN ('internal','external','online','workshop','conference','certification','mentoring')),
    status              VARCHAR(20) NOT NULL DEFAULT 'planned'
                        CHECK (status IN ('planned','registration_open','in_progress','completed','cancelled')),
    provider            VARCHAR(255),
    trainer             VARCHAR(255),
    start_date          DATE NOT NULL,
    end_date            DATE NOT NULL,
    location            TEXT,
    max_participants    INTEGER NOT NULL DEFAULT 0,
    cost                DECIMAL(12,2),
    mandatory           BOOLEAN NOT NULL DEFAULT FALSE,
    department_ids      JSONB NOT NULL DEFAULT '[]',
    created_by          BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. training_participants
CREATE TABLE training_participants (
    id              BIGSERIAL PRIMARY KEY,
    training_id     BIGINT NOT NULL REFERENCES trainings(id) ON DELETE CASCADE,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL DEFAULT 'enrolled'
                    CHECK (status IN ('enrolled','attending','completed','cancelled','no_show')),
    enrolled_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    score           INTEGER CHECK (score IS NULL OR (score >= 0 AND score <= 100)),
    certificate_id  BIGINT,
    notes           TEXT,
    UNIQUE(training_id, employee_id)
);

-- 3. certificates
CREATE TABLE certificates (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    training_id     BIGINT REFERENCES trainings(id) ON DELETE SET NULL,
    title           VARCHAR(255) NOT NULL,
    issuer          VARCHAR(255),
    issue_date      DATE NOT NULL,
    expiry_date     DATE,
    certificate_url TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. development_plans
CREATE TABLE development_plans (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','active','completed','cancelled')),
    start_date      DATE,
    end_date        DATE,
    created_by      BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. development_goals
CREATE TABLE development_goals (
    id                  BIGSERIAL PRIMARY KEY,
    plan_id             BIGINT NOT NULL REFERENCES development_plans(id) ON DELETE CASCADE,
    title               VARCHAR(255) NOT NULL,
    description         TEXT,
    status              VARCHAR(20) NOT NULL DEFAULT 'not_started'
                        CHECK (status IN ('not_started','in_progress','completed','cancelled')),
    target_date         DATE,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FK: training_participants.certificate_id -> certificates
ALTER TABLE training_participants
    ADD CONSTRAINT fk_training_participants_certificate
    FOREIGN KEY (certificate_id) REFERENCES certificates(id) ON DELETE SET NULL;

-- Indexes
CREATE INDEX idx_trainings_status_type ON trainings(status, type);
CREATE INDEX idx_training_participants_training_employee ON training_participants(training_id, employee_id, status);
CREATE INDEX idx_certificates_employee ON certificates(employee_id);
CREATE INDEX idx_development_plans_employee_status ON development_plans(employee_id, status);
CREATE INDEX idx_development_goals_plan_status ON development_goals(plan_id, status);

-- Triggers for updated_at
CREATE TRIGGER set_timestamp_trainings
    BEFORE UPDATE ON trainings
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_development_plans
    BEFORE UPDATE ON development_plans
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
