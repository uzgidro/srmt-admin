-- Справочник компетенций
CREATE TABLE competencies (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    category    VARCHAR(20) NOT NULL DEFAULT 'professional'
                CHECK (category IN ('professional','personal','managerial','technical','communication','leadership')),
    levels      JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Привязка компетенций к должностям (M:N)
CREATE TABLE competency_positions (
    id              BIGSERIAL PRIMARY KEY,
    competency_id   BIGINT NOT NULL REFERENCES competencies(id) ON DELETE CASCADE,
    position_id     BIGINT NOT NULL REFERENCES positions(id) ON DELETE CASCADE,
    required_level  INTEGER NOT NULL DEFAULT 1 CHECK (required_level BETWEEN 1 AND 5),
    UNIQUE(competency_id, position_id)
);

-- Сессии ассессмента
CREATE TABLE assessment_sessions (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    status      VARCHAR(20) NOT NULL DEFAULT 'draft'
                CHECK (status IN ('draft','planned','in_progress','completed','cancelled')),
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,
    created_by  BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Компетенции сессии (какие компетенции оцениваются, с весами)
CREATE TABLE assessment_competencies (
    id             BIGSERIAL PRIMARY KEY,
    session_id     BIGINT NOT NULL REFERENCES assessment_sessions(id) ON DELETE CASCADE,
    competency_id  BIGINT NOT NULL REFERENCES competencies(id) ON DELETE CASCADE,
    weight         NUMERIC(5,2) NOT NULL DEFAULT 1.0,
    required_level INTEGER NOT NULL DEFAULT 1 CHECK (required_level BETWEEN 1 AND 5),
    UNIQUE(session_id, competency_id)
);

-- Кандидаты (оцениваемые сотрудники) сессии
CREATE TABLE assessment_candidates (
    id          BIGSERIAL PRIMARY KEY,
    session_id  BIGINT NOT NULL REFERENCES assessment_sessions(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','in_progress','completed')),
    UNIQUE(session_id, employee_id)
);

-- Оценщики (эксперты) сессии
CREATE TABLE assessment_assessors (
    id          BIGSERIAL PRIMARY KEY,
    session_id  BIGINT NOT NULL REFERENCES assessment_sessions(id) ON DELETE CASCADE,
    employee_id BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    role        VARCHAR(20) NOT NULL DEFAULT 'expert'
                CHECK (role IN ('manager','peer','self','expert','subordinate')),
    UNIQUE(session_id, employee_id)
);

-- Оценки (кто-кому-какую компетенцию-на-какой-уровень)
CREATE TABLE assessment_scores (
    id              BIGSERIAL PRIMARY KEY,
    session_id      BIGINT NOT NULL REFERENCES assessment_sessions(id) ON DELETE CASCADE,
    candidate_id    BIGINT NOT NULL REFERENCES assessment_candidates(id) ON DELETE CASCADE,
    assessor_id     BIGINT NOT NULL REFERENCES assessment_assessors(id) ON DELETE CASCADE,
    competency_id   BIGINT NOT NULL REFERENCES competencies(id) ON DELETE CASCADE,
    score           INTEGER NOT NULL CHECK (score BETWEEN 1 AND 5),
    comment         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(session_id, candidate_id, assessor_id, competency_id)
);

-- Indexes
CREATE INDEX idx_competencies_category ON competencies(category);
CREATE INDEX idx_competency_positions_position_id ON competency_positions(position_id);
CREATE INDEX idx_assessment_sessions_status ON assessment_sessions(status);
CREATE INDEX idx_assessment_competencies_session_id ON assessment_competencies(session_id);
CREATE INDEX idx_assessment_candidates_session_employee ON assessment_candidates(session_id, employee_id);
CREATE INDEX idx_assessment_assessors_session_id ON assessment_assessors(session_id);
CREATE INDEX idx_assessment_scores_session_candidate ON assessment_scores(session_id, candidate_id);

-- Triggers
CREATE TRIGGER set_timestamp_competencies
    BEFORE UPDATE ON competencies
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_assessment_sessions
    BEFORE UPDATE ON assessment_sessions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
