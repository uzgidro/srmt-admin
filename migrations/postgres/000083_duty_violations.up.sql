-- Прогулы дежурных (duty-officer violations).
-- Org-scoped CRUD records with multiple file attachments.

CREATE TABLE duty_violations (
    id                  BIGSERIAL PRIMARY KEY,
    organization_id     BIGINT NOT NULL
                        REFERENCES organizations(id) ON DELETE RESTRICT,
    start_time          TIMESTAMPTZ NOT NULL,
    end_time            TIMESTAMPTZ NOT NULL,
    duty_officer_name   TEXT NOT NULL,
    reason              TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT duty_violations_time_range CHECK (end_time > start_time),
    CONSTRAINT duty_violations_name_not_blank CHECK (length(trim(duty_officer_name)) > 0),
    CONSTRAINT duty_violations_reason_not_blank CHECK (length(trim(reason)) > 0)
);

CREATE INDEX idx_duty_violations_org_time
    ON duty_violations(organization_id, start_time DESC);
CREATE INDEX idx_duty_violations_start_time
    ON duty_violations(start_time DESC);

CREATE TRIGGER set_timestamp_duty_violations
    BEFORE UPDATE ON duty_violations
    FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- Junction table mirrors incident_file_links: per-record list of attached files.
CREATE TABLE duty_violation_file_links (
    duty_violation_id BIGINT NOT NULL
                      REFERENCES duty_violations(id) ON DELETE CASCADE,
    file_id           BIGINT NOT NULL
                      REFERENCES files(id) ON DELETE CASCADE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (duty_violation_id, file_id)
);
CREATE INDEX idx_duty_violation_file_links_dv
    ON duty_violation_file_links(duty_violation_id);
CREATE INDEX idx_duty_violation_file_links_file
    ON duty_violation_file_links(file_id);
