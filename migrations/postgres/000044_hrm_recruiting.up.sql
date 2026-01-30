-- HRM Recruiting Management

-- Job vacancies
CREATE TABLE IF NOT EXISTS hrm_vacancies (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    position_id BIGINT REFERENCES positions(id) ON DELETE SET NULL,
    department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,

    -- Details
    description TEXT,
    requirements TEXT,
    responsibilities TEXT,
    benefits TEXT,

    -- Employment terms
    employment_type VARCHAR(50) DEFAULT 'full_time', -- full_time, part_time, contract, intern
    work_format VARCHAR(50) DEFAULT 'office', -- office, remote, hybrid
    experience_level VARCHAR(50), -- junior, middle, senior, lead

    -- Salary range
    salary_min DECIMAL(15,2),
    salary_max DECIMAL(15,2),
    currency VARCHAR(3) DEFAULT 'RUB',
    salary_visible BOOLEAN DEFAULT FALSE,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, open, paused, closed, filled
    priority VARCHAR(20) DEFAULT 'normal', -- low, normal, high, urgent
    headcount INTEGER DEFAULT 1,
    filled_count INTEGER DEFAULT 0,

    -- Dates
    published_at TIMESTAMPTZ,
    deadline DATE,
    closed_at TIMESTAMPTZ,

    -- Responsible
    hiring_manager_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,
    recruiter_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_vacancies_status ON hrm_vacancies(status);
CREATE INDEX idx_hrm_vacancies_department ON hrm_vacancies(department_id);
CREATE INDEX idx_hrm_vacancies_position ON hrm_vacancies(position_id);
CREATE INDEX idx_hrm_vacancies_hiring_manager ON hrm_vacancies(hiring_manager_id);
CREATE INDEX idx_hrm_vacancies_recruiter ON hrm_vacancies(recruiter_id);

CREATE TRIGGER set_timestamp_hrm_vacancies
    BEFORE UPDATE ON hrm_vacancies
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Job candidates
CREATE TABLE IF NOT EXISTS hrm_candidates (
    id BIGSERIAL PRIMARY KEY,
    vacancy_id BIGINT NOT NULL REFERENCES hrm_vacancies(id) ON DELETE CASCADE,

    -- Personal info
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    email VARCHAR(255),
    phone VARCHAR(50),

    -- Professional info
    current_position VARCHAR(255),
    current_company VARCHAR(255),
    experience_years INTEGER,
    expected_salary DECIMAL(15,2),
    currency VARCHAR(3) DEFAULT 'RUB',

    -- Documents
    resume_file_id BIGINT REFERENCES files(id) ON DELETE SET NULL,
    cover_letter TEXT,

    -- Source
    source VARCHAR(100), -- hh.ru, linkedin, referral, website, etc.
    referrer_employee_id BIGINT REFERENCES hrm_employees(id) ON DELETE SET NULL,

    -- Pipeline status
    status VARCHAR(30) NOT NULL DEFAULT 'new', -- new, screening, interview, offer, hired, rejected, withdrawn
    rejection_reason TEXT,

    -- Ratings
    rating INTEGER CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5)),
    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_candidates_vacancy ON hrm_candidates(vacancy_id);
CREATE INDEX idx_hrm_candidates_status ON hrm_candidates(status);
CREATE INDEX idx_hrm_candidates_email ON hrm_candidates(email);
CREATE INDEX idx_hrm_candidates_source ON hrm_candidates(source);

CREATE TRIGGER set_timestamp_hrm_candidates
    BEFORE UPDATE ON hrm_candidates
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Interview records
CREATE TABLE IF NOT EXISTS hrm_interviews (
    id BIGSERIAL PRIMARY KEY,
    candidate_id BIGINT NOT NULL REFERENCES hrm_candidates(id) ON DELETE CASCADE,
    vacancy_id BIGINT NOT NULL REFERENCES hrm_vacancies(id) ON DELETE CASCADE,

    -- Interview details
    interview_type VARCHAR(50) NOT NULL, -- phone, video, onsite, technical, hr, final
    scheduled_at TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER DEFAULT 60,
    location VARCHAR(255), -- Room or video link

    -- Interviewers
    interviewer_ids BIGINT[] NOT NULL DEFAULT '{}',

    -- Results
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled', -- scheduled, completed, cancelled, no_show
    completed_at TIMESTAMPTZ,
    overall_rating INTEGER CHECK (overall_rating IS NULL OR (overall_rating >= 1 AND overall_rating <= 5)),
    feedback TEXT,
    recommendation VARCHAR(20), -- hire, maybe, reject

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_interviews_candidate ON hrm_interviews(candidate_id);
CREATE INDEX idx_hrm_interviews_vacancy ON hrm_interviews(vacancy_id);
CREATE INDEX idx_hrm_interviews_scheduled ON hrm_interviews(scheduled_at);
CREATE INDEX idx_hrm_interviews_status ON hrm_interviews(status);

CREATE TRIGGER set_timestamp_hrm_interviews
    BEFORE UPDATE ON hrm_interviews
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_vacancies IS 'Job vacancies/openings';
COMMENT ON TABLE hrm_candidates IS 'Job applicants/candidates';
COMMENT ON TABLE hrm_interviews IS 'Interview records for candidates';
