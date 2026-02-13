-- Performance Reviews
CREATE TABLE IF NOT EXISTS performance_reviews (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    reviewer_id     BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    type            VARCHAR(20) NOT NULL DEFAULT 'annual'
                    CHECK (type IN ('annual','quarterly','probation','project','mid_year')),
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','self_review','manager_review','calibration','completed','acknowledged')),
    period_start    DATE NOT NULL,
    period_end      DATE NOT NULL,
    self_rating     INTEGER CHECK (self_rating BETWEEN 1 AND 5),
    manager_rating  INTEGER CHECK (manager_rating BETWEEN 1 AND 5),
    final_rating    INTEGER CHECK (final_rating BETWEEN 1 AND 5),
    self_comment    TEXT,
    manager_comment TEXT,
    strengths       TEXT,
    improvements    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Performance Goals
CREATE TABLE IF NOT EXISTS performance_goals (
    id              BIGSERIAL PRIMARY KEY,
    review_id       BIGINT REFERENCES performance_reviews(id) ON DELETE CASCADE,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    metric          VARCHAR(255),
    target_value    NUMERIC(12,2) NOT NULL DEFAULT 0,
    current_value   NUMERIC(12,2) NOT NULL DEFAULT 0,
    weight          NUMERIC(5,2) NOT NULL DEFAULT 1.0,
    status          VARCHAR(20) NOT NULL DEFAULT 'not_started'
                    CHECK (status IN ('not_started','in_progress','completed','overdue','cancelled')),
    due_date        DATE NOT NULL,
    progress        INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_performance_reviews_employee ON performance_reviews(employee_id);
CREATE INDEX IF NOT EXISTS idx_performance_reviews_status ON performance_reviews(status);
CREATE INDEX IF NOT EXISTS idx_performance_reviews_type ON performance_reviews(type);
CREATE INDEX IF NOT EXISTS idx_performance_goals_review ON performance_goals(review_id);
CREATE INDEX IF NOT EXISTS idx_performance_goals_employee ON performance_goals(employee_id);
CREATE INDEX IF NOT EXISTS idx_performance_goals_status ON performance_goals(status);

-- Triggers for updated_at
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'set_timestamp_performance_reviews') THEN
        CREATE TRIGGER set_timestamp_performance_reviews
            BEFORE UPDATE ON performance_reviews
            FOR EACH ROW
            EXECUTE FUNCTION trigger_set_timestamp();
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'set_timestamp_performance_goals') THEN
        CREATE TRIGGER set_timestamp_performance_goals
            BEFORE UPDATE ON performance_goals
            FOR EACH ROW
            EXECUTE FUNCTION trigger_set_timestamp();
    END IF;
END $$;
