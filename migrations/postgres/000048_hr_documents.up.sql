-- HR Documents
CREATE TABLE hr_documents (
    id              BIGSERIAL PRIMARY KEY,
    title           VARCHAR(255) NOT NULL,
    type            VARCHAR(20) NOT NULL DEFAULT 'other'
                    CHECK (type IN ('order','contract','agreement','certificate','reference',
                                    'memo','report','protocol','regulation','instruction','application','other')),
    category        VARCHAR(20) NOT NULL DEFAULT 'other'
                    CHECK (category IN ('personnel','administrative','financial','legal','other')),
    number          VARCHAR(100) NOT NULL,
    date            DATE NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','pending_review','pending_signatures','active','archived','cancelled')),
    content         TEXT,
    file_url        TEXT,
    department_id   BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    employee_id     BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    created_by      BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE document_signatures (
    id              BIGSERIAL PRIMARY KEY,
    document_id     BIGINT NOT NULL REFERENCES hr_documents(id) ON DELETE CASCADE,
    signer_id       BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','signed','rejected')),
    signed_at       TIMESTAMPTZ,
    comment         TEXT,
    sign_order      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE document_requests (
    id              BIGSERIAL PRIMARY KEY,
    employee_id     BIGINT NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    document_type   VARCHAR(50) NOT NULL,
    purpose         TEXT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','in_progress','ready','rejected','delivered')),
    rejection_reason TEXT,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_hr_documents_status_type_category ON hr_documents(status, type, category);
CREATE INDEX idx_document_signatures_document_id ON document_signatures(document_id);
CREATE INDEX idx_document_requests_employee_status ON document_requests(employee_id, status);

CREATE TRIGGER set_timestamp_hr_documents
    BEFORE UPDATE ON hr_documents
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

CREATE TRIGGER set_timestamp_document_requests
    BEFORE UPDATE ON document_requests
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();
