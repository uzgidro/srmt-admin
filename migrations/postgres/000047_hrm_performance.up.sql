-- HRM Performance Management

-- Performance review cycles
CREATE TABLE IF NOT EXISTS hrm_performance_reviews (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- Review period
    review_type VARCHAR(50) NOT NULL, -- annual, mid_year, quarterly, probation
    review_period_start DATE NOT NULL,
    review_period_end DATE NOT NULL,

    -- Status and workflow
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, self_review, manager_review, calibration, completed
    self_review_deadline DATE,
    manager_review_deadline DATE,

    -- Self review
    self_review_started_at TIMESTAMPTZ,
    self_review_completed_at TIMESTAMPTZ,
    self_assessment TEXT,
    self_rating INTEGER CHECK (self_rating IS NULL OR (self_rating >= 1 AND self_rating <= 5)),

    -- Manager review
    reviewer_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,
    manager_review_started_at TIMESTAMPTZ,
    manager_review_completed_at TIMESTAMPTZ,
    manager_assessment TEXT,
    manager_rating INTEGER CHECK (manager_rating IS NULL OR (manager_rating >= 1 AND manager_rating <= 5)),

    -- Final rating
    final_rating INTEGER CHECK (final_rating IS NULL OR (final_rating >= 1 AND final_rating <= 5)),
    final_rating_label VARCHAR(50), -- e.g., "Exceeds Expectations", "Meets Expectations"
    calibrated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    calibrated_at TIMESTAMPTZ,

    -- Summary
    achievements TEXT,
    areas_for_improvement TEXT,
    development_recommendations TEXT,

    completed_at TIMESTAMPTZ,
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_performance_reviews_employee ON hrm_performance_reviews(employee_id);
CREATE INDEX idx_hrm_performance_reviews_reviewer ON hrm_performance_reviews(reviewer_id);
CREATE INDEX idx_hrm_performance_reviews_status ON hrm_performance_reviews(status);
CREATE INDEX idx_hrm_performance_reviews_type ON hrm_performance_reviews(review_type);
CREATE INDEX idx_hrm_performance_reviews_period ON hrm_performance_reviews(review_period_start, review_period_end);

CREATE TRIGGER set_timestamp_hrm_performance_reviews
    BEFORE UPDATE ON hrm_performance_reviews
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Performance goals (OKRs, objectives)
CREATE TABLE IF NOT EXISTS hrm_performance_goals (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,
    review_id BIGINT REFERENCES hrm_performance_reviews(id) ON DELETE SET NULL,

    -- Goal details
    title VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50), -- business, development, team, process

    -- SMART criteria
    success_criteria TEXT,
    metrics TEXT,

    -- Alignment
    aligned_to VARCHAR(255), -- Company/team objective this aligns to
    weight DECIMAL(3,2) DEFAULT 1.0, -- Weight for goal scoring

    -- Timeline
    start_date DATE,
    target_date DATE,

    -- Progress
    status VARCHAR(20) NOT NULL DEFAULT 'not_started', -- not_started, in_progress, completed, cancelled, deferred
    progress INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),

    -- Self assessment
    self_rating INTEGER CHECK (self_rating IS NULL OR (self_rating >= 1 AND self_rating <= 5)),
    self_comments TEXT,

    -- Manager assessment
    manager_rating INTEGER CHECK (manager_rating IS NULL OR (manager_rating >= 1 AND manager_rating <= 5)),
    manager_comments TEXT,

    completed_at TIMESTAMPTZ,
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_performance_goals_employee ON hrm_performance_goals(employee_id);
CREATE INDEX idx_hrm_performance_goals_review ON hrm_performance_goals(review_id);
CREATE INDEX idx_hrm_performance_goals_status ON hrm_performance_goals(status);
CREATE INDEX idx_hrm_performance_goals_dates ON hrm_performance_goals(start_date, target_date);

CREATE TRIGGER set_timestamp_hrm_performance_goals
    BEFORE UPDATE ON hrm_performance_goals
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- KPI tracking
CREATE TABLE IF NOT EXISTS hrm_kpis (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- KPI definition
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100), -- sales, quality, productivity, customer, etc.

    -- Measurement
    measurement_unit VARCHAR(50), -- percent, count, currency, score
    target_value DECIMAL(15,2) NOT NULL,
    min_threshold DECIMAL(15,2), -- Minimum acceptable value
    max_threshold DECIMAL(15,2), -- Stretch goal

    -- Period
    year INTEGER NOT NULL,
    month INTEGER CHECK (month IS NULL OR (month >= 1 AND month <= 12)),
    quarter INTEGER CHECK (quarter IS NULL OR (quarter >= 1 AND quarter <= 4)),

    -- Actual results
    actual_value DECIMAL(15,2),
    achievement_percent DECIMAL(5,2), -- (actual/target) * 100

    -- Rating
    rating INTEGER CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5)),

    -- Weight for overall scoring
    weight DECIMAL(3,2) DEFAULT 1.0,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_kpis_employee ON hrm_kpis(employee_id);
CREATE INDEX idx_hrm_kpis_period ON hrm_kpis(year, quarter, month);
CREATE INDEX idx_hrm_kpis_category ON hrm_kpis(category);

CREATE TRIGGER set_timestamp_hrm_kpis
    BEFORE UPDATE ON hrm_kpis
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_performance_reviews IS 'Performance review cycles (annual, quarterly)';
COMMENT ON TABLE hrm_performance_goals IS 'Performance goals and objectives';
COMMENT ON TABLE hrm_kpis IS 'Key Performance Indicators tracking';
