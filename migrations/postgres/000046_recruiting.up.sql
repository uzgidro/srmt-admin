-- 1. Vacancies
CREATE TABLE vacancies (
    id                  BIGSERIAL PRIMARY KEY,
    title               VARCHAR(255) NOT NULL,
    department_id       BIGINT NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
    position_id         BIGINT NOT NULL REFERENCES positions(id) ON DELETE RESTRICT,
    description         TEXT,
    requirements        TEXT,
    salary_from         DECIMAL(12,2),
    salary_to           DECIMAL(12,2),
    employment_type     VARCHAR(20) NOT NULL DEFAULT 'full_time'
                        CHECK (employment_type IN ('full_time','part_time','contract','internship')),
    experience_required VARCHAR(100),
    education_required  VARCHAR(100),
    skills              JSONB NOT NULL DEFAULT '[]',
    status              VARCHAR(20) NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft','pending_approval','approved','published','closed','cancelled','on_hold')),
    priority            VARCHAR(10) NOT NULL DEFAULT 'medium'
                        CHECK (priority IN ('low','medium','high','urgent')),
    published_at        TIMESTAMPTZ,
    deadline            DATE,
    responsible_id      BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    created_by          BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vacancies_department_id ON vacancies(department_id);
CREATE INDEX idx_vacancies_status ON vacancies(status);
CREATE INDEX idx_vacancies_priority ON vacancies(priority);

CREATE TRIGGER set_timestamp_vacancies
    BEFORE UPDATE ON vacancies
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- 2. Candidates
CREATE TABLE candidates (
    id                  BIGSERIAL PRIMARY KEY,
    vacancy_id          BIGINT NOT NULL REFERENCES vacancies(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    email               VARCHAR(255),
    phone               VARCHAR(50),
    source              VARCHAR(20) NOT NULL DEFAULT 'other'
                        CHECK (source IN ('website','linkedin','referral','agency','job_board','social_media','university','internal','other')),
    status              VARCHAR(20) NOT NULL DEFAULT 'new'
                        CHECK (status IN ('new','screening','phone_interview','assessment','interview','offer','hired','rejected','withdrawn','blacklisted')),
    stage               VARCHAR(20) NOT NULL DEFAULT 'sourcing'
                        CHECK (stage IN ('sourcing','screening','interview','assessment','offer','hiring','onboarding')),
    resume_url          TEXT,
    photo_url           TEXT,
    skills              JSONB NOT NULL DEFAULT '[]',
    languages           JSONB NOT NULL DEFAULT '[]',
    salary_expectation  DECIMAL(12,2),
    notes               TEXT,
    rating              INTEGER CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5)),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_candidates_vacancy_id ON candidates(vacancy_id);
CREATE INDEX idx_candidates_status ON candidates(status);
CREATE INDEX idx_candidates_stage ON candidates(stage);
CREATE INDEX idx_candidates_email ON candidates(email);

CREATE TRIGGER set_timestamp_candidates
    BEFORE UPDATE ON candidates
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- 3. Candidate Education
CREATE TABLE candidate_education (
    id              BIGSERIAL PRIMARY KEY,
    candidate_id    BIGINT NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    institution     VARCHAR(255) NOT NULL,
    degree          VARCHAR(100),
    field_of_study  VARCHAR(255),
    start_date      DATE,
    end_date        DATE,
    description     TEXT
);

CREATE INDEX idx_candidate_education_candidate_id ON candidate_education(candidate_id);

-- 4. Candidate Experience
CREATE TABLE candidate_experience (
    id              BIGSERIAL PRIMARY KEY,
    candidate_id    BIGINT NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    company         VARCHAR(255) NOT NULL,
    position        VARCHAR(255),
    start_date      DATE,
    end_date        DATE,
    description     TEXT
);

CREATE INDEX idx_candidate_experience_candidate_id ON candidate_experience(candidate_id);

-- 5. Interviews
CREATE TABLE interviews (
    id                  BIGSERIAL PRIMARY KEY,
    candidate_id        BIGINT NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    vacancy_id          BIGINT NOT NULL REFERENCES vacancies(id) ON DELETE CASCADE,
    type                VARCHAR(20) NOT NULL DEFAULT 'hr'
                        CHECK (type IN ('phone','video','in_person','technical','hr','final','group')),
    scheduled_at        TIMESTAMPTZ NOT NULL,
    duration_minutes    INTEGER NOT NULL DEFAULT 60,
    location            TEXT,
    interviewers        JSONB NOT NULL DEFAULT '[]',
    status              VARCHAR(20) NOT NULL DEFAULT 'scheduled'
                        CHECK (status IN ('scheduled','in_progress','completed','cancelled','no_show','rescheduled')),
    overall_rating      INTEGER CHECK (overall_rating IS NULL OR (overall_rating >= 1 AND overall_rating <= 5)),
    recommendation      VARCHAR(20)
                        CHECK (recommendation IS NULL OR recommendation IN ('strong_hire','hire','no_hire','strong_no_hire')),
    feedback            TEXT,
    scores              JSONB NOT NULL DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_interviews_candidate_id ON interviews(candidate_id);
CREATE INDEX idx_interviews_vacancy_id ON interviews(vacancy_id);
CREATE INDEX idx_interviews_status ON interviews(status);
CREATE INDEX idx_interviews_scheduled_at ON interviews(scheduled_at);

CREATE TRIGGER set_timestamp_interviews
    BEFORE UPDATE ON interviews
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- 6. Job Offers (stub)
CREATE TABLE job_offers (
    id              BIGSERIAL PRIMARY KEY,
    candidate_id    BIGINT NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    vacancy_id      BIGINT NOT NULL REFERENCES vacancies(id) ON DELETE CASCADE,
    salary_offered  DECIMAL(12,2),
    start_date      DATE,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','sent','accepted','rejected','expired','withdrawn')),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_timestamp_job_offers
    BEFORE UPDATE ON job_offers
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- 7. Onboardings (stub)
CREATE TABLE onboardings (
    id              BIGSERIAL PRIMARY KEY,
    candidate_id    BIGINT NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    vacancy_id      BIGINT NOT NULL REFERENCES vacancies(id) ON DELETE CASCADE,
    start_date      DATE,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','in_progress','completed','cancelled')),
    mentor_id       BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_timestamp_onboardings
    BEFORE UPDATE ON onboardings
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- 8. Onboarding Tasks (stub)
CREATE TABLE onboarding_tasks (
    id              BIGSERIAL PRIMARY KEY,
    onboarding_id   BIGINT NOT NULL REFERENCES onboardings(id) ON DELETE CASCADE,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    assigned_to     BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    due_date        DATE,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','in_progress','completed','skipped')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_timestamp_onboarding_tasks
    BEFORE UPDATE ON onboarding_tasks
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
