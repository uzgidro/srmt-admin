-- HRM Competency Management

-- Competency categories
CREATE TABLE IF NOT EXISTS hrm_competency_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    sort_order INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE TRIGGER set_timestamp_hrm_competency_categories
    BEFORE UPDATE ON hrm_competency_categories
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Insert default categories
INSERT INTO hrm_competency_categories (name, description, sort_order) VALUES
    ('Technical Skills', 'Job-specific technical competencies', 1),
    ('Soft Skills', 'Communication, teamwork, and interpersonal skills', 2),
    ('Leadership', 'Management and leadership competencies', 3),
    ('Business', 'Business acumen and strategic thinking', 4),
    ('Core Values', 'Company culture and values alignment', 5);

-- Competencies
CREATE TABLE IF NOT EXISTS hrm_competencies (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES hrm_competency_categories(id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(20) UNIQUE,
    description TEXT,
    behavioral_indicators TEXT, -- Examples of demonstrating competency

    is_active BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_competency_name_category UNIQUE (category_id, name)
);

CREATE INDEX idx_hrm_competencies_category ON hrm_competencies(category_id);
CREATE INDEX idx_hrm_competencies_active ON hrm_competencies(is_active) WHERE is_active = TRUE;

CREATE TRIGGER set_timestamp_hrm_competencies
    BEFORE UPDATE ON hrm_competencies
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Competency levels (proficiency scale)
CREATE TABLE IF NOT EXISTS hrm_competency_levels (
    id SERIAL PRIMARY KEY,
    competency_id INTEGER NOT NULL REFERENCES hrm_competencies(id) ON DELETE CASCADE,

    level INTEGER NOT NULL CHECK (level >= 1 AND level <= 5),
    name VARCHAR(50) NOT NULL, -- e.g., Beginner, Developing, Proficient, Advanced, Expert
    description TEXT, -- What this level means for this competency

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_competency_level UNIQUE (competency_id, level)
);

CREATE INDEX idx_hrm_competency_levels_competency ON hrm_competency_levels(competency_id);

CREATE TRIGGER set_timestamp_hrm_competency_levels
    BEFORE UPDATE ON hrm_competency_levels
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Competency matrices (required levels per position)
CREATE TABLE IF NOT EXISTS hrm_competency_matrices (
    id BIGSERIAL PRIMARY KEY,
    position_id BIGINT NOT NULL REFERENCES positions(id) ON DELETE CASCADE,
    competency_id INTEGER NOT NULL REFERENCES hrm_competencies(id) ON DELETE CASCADE,

    required_level INTEGER NOT NULL CHECK (required_level >= 1 AND required_level <= 5),
    is_mandatory BOOLEAN DEFAULT FALSE,
    weight DECIMAL(3,2) DEFAULT 1.0, -- For weighted scoring

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_position_competency UNIQUE (position_id, competency_id)
);

CREATE INDEX idx_hrm_competency_matrices_position ON hrm_competency_matrices(position_id);
CREATE INDEX idx_hrm_competency_matrices_competency ON hrm_competency_matrices(competency_id);

CREATE TRIGGER set_timestamp_hrm_competency_matrices
    BEFORE UPDATE ON hrm_competency_matrices
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Competency assessments (assessment sessions)
CREATE TABLE IF NOT EXISTS hrm_competency_assessments (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    assessment_type VARCHAR(50) NOT NULL, -- self, manager, peer, 360
    assessment_period_start DATE NOT NULL,
    assessment_period_end DATE NOT NULL,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, in_progress, completed
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Assessor
    assessor_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,

    -- Results
    overall_score DECIMAL(3,1),
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_competency_assessments_employee ON hrm_competency_assessments(employee_id);
CREATE INDEX idx_hrm_competency_assessments_assessor ON hrm_competency_assessments(assessor_id);
CREATE INDEX idx_hrm_competency_assessments_status ON hrm_competency_assessments(status);
CREATE INDEX idx_hrm_competency_assessments_period ON hrm_competency_assessments(assessment_period_start, assessment_period_end);

CREATE TRIGGER set_timestamp_hrm_competency_assessments
    BEFORE UPDATE ON hrm_competency_assessments
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Individual competency scores
CREATE TABLE IF NOT EXISTS hrm_competency_scores (
    id BIGSERIAL PRIMARY KEY,
    assessment_id BIGINT NOT NULL REFERENCES hrm_competency_assessments(id) ON DELETE CASCADE,
    competency_id INTEGER NOT NULL REFERENCES hrm_competencies(id) ON DELETE RESTRICT,

    score INTEGER NOT NULL CHECK (score >= 1 AND score <= 5),
    evidence TEXT, -- Supporting examples/evidence
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_assessment_competency UNIQUE (assessment_id, competency_id)
);

CREATE INDEX idx_hrm_competency_scores_assessment ON hrm_competency_scores(assessment_id);
CREATE INDEX idx_hrm_competency_scores_competency ON hrm_competency_scores(competency_id);

CREATE TRIGGER set_timestamp_hrm_competency_scores
    BEFORE UPDATE ON hrm_competency_scores
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_competency_categories IS 'Categories of competencies (technical, soft skills, etc.)';
COMMENT ON TABLE hrm_competencies IS 'Competency definitions';
COMMENT ON TABLE hrm_competency_levels IS 'Proficiency levels for each competency';
COMMENT ON TABLE hrm_competency_matrices IS 'Required competency levels per position';
COMMENT ON TABLE hrm_competency_assessments IS 'Competency assessment sessions';
COMMENT ON TABLE hrm_competency_scores IS 'Individual competency scores in assessments';
