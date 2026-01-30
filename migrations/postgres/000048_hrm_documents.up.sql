-- HRM Document Management

-- HR document types
CREATE TABLE IF NOT EXISTS hrm_document_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(30) UNIQUE,
    description TEXT,

    -- Template
    template_id BIGINT REFERENCES files(id) ON DELETE SET NULL,

    -- Configuration
    requires_signature BOOLEAN DEFAULT FALSE,
    requires_employee_signature BOOLEAN DEFAULT FALSE,
    requires_manager_signature BOOLEAN DEFAULT FALSE,
    requires_hr_signature BOOLEAN DEFAULT FALSE,

    expiry_days INTEGER, -- Days until document expires (NULL = no expiry)

    is_active BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE TRIGGER set_timestamp_hrm_document_types
    BEFORE UPDATE ON hrm_document_types
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Insert default document types
INSERT INTO hrm_document_types (name, code, requires_signature, requires_employee_signature, sort_order) VALUES
    ('Employment Contract', 'CONTRACT', TRUE, TRUE, 1),
    ('NDA', 'NDA', TRUE, TRUE, 2),
    ('Job Description', 'JOB_DESC', FALSE, FALSE, 3),
    ('Policy Acknowledgment', 'POLICY', TRUE, TRUE, 4),
    ('Performance Review', 'PERF_REVIEW', TRUE, TRUE, 5),
    ('Termination Letter', 'TERMINATION', TRUE, TRUE, 6),
    ('Promotion Letter', 'PROMOTION', TRUE, FALSE, 7),
    ('Warning Letter', 'WARNING', TRUE, TRUE, 8),
    ('Training Certificate', 'TRAINING_CERT', FALSE, FALSE, 9),
    ('Leave Request', 'LEAVE', FALSE, FALSE, 10);

-- HR documents
CREATE TABLE IF NOT EXISTS hrm_documents (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,
    document_type_id INTEGER NOT NULL REFERENCES hrm_document_types(id) ON DELETE RESTRICT,

    -- Document info
    title VARCHAR(255) NOT NULL,
    document_number VARCHAR(100),
    description TEXT,

    -- File
    file_id BIGINT REFERENCES files(id) ON DELETE SET NULL,

    -- Dates
    issue_date DATE,
    effective_date DATE,
    expiry_date DATE,

    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'draft', -- draft, pending_signature, active, expired, archived

    -- Created by
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_documents_employee ON hrm_documents(employee_id);
CREATE INDEX idx_hrm_documents_type ON hrm_documents(document_type_id);
CREATE INDEX idx_hrm_documents_status ON hrm_documents(status);
CREATE INDEX idx_hrm_documents_expiry ON hrm_documents(expiry_date);

CREATE TRIGGER set_timestamp_hrm_documents
    BEFORE UPDATE ON hrm_documents
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Document signatures
CREATE TABLE IF NOT EXISTS hrm_document_signatures (
    id BIGSERIAL PRIMARY KEY,
    document_id BIGINT NOT NULL REFERENCES hrm_documents(id) ON DELETE CASCADE,

    -- Signer
    signer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    signer_role VARCHAR(50) NOT NULL, -- employee, manager, hr, director

    -- Signature
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, signed, rejected
    signed_at TIMESTAMPTZ,
    signature_ip VARCHAR(45),
    rejection_reason TEXT,

    -- Order
    sign_order INTEGER DEFAULT 0, -- For sequential signing

    notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,

    CONSTRAINT unique_document_signer UNIQUE (document_id, signer_user_id)
);

CREATE INDEX idx_hrm_document_signatures_document ON hrm_document_signatures(document_id);
CREATE INDEX idx_hrm_document_signatures_signer ON hrm_document_signatures(signer_user_id);
CREATE INDEX idx_hrm_document_signatures_status ON hrm_document_signatures(status);

CREATE TRIGGER set_timestamp_hrm_document_signatures
    BEFORE UPDATE ON hrm_document_signatures
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Document templates
CREATE TABLE IF NOT EXISTS hrm_document_templates (
    id BIGSERIAL PRIMARY KEY,
    document_type_id INTEGER NOT NULL REFERENCES hrm_document_types(id) ON DELETE RESTRICT,

    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Template content
    content TEXT, -- HTML/Markdown template with placeholders
    file_id BIGINT REFERENCES files(id) ON DELETE SET NULL, -- DOCX/PDF template

    -- Placeholders available (for UI)
    placeholders JSONB DEFAULT '[]', -- [{key: "employee_name", label: "Employee Name"}, ...]

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    version INTEGER DEFAULT 1,

    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_document_templates_type ON hrm_document_templates(document_type_id);
CREATE INDEX idx_hrm_document_templates_active ON hrm_document_templates(is_active) WHERE is_active = TRUE;

CREATE TRIGGER set_timestamp_hrm_document_templates
    BEFORE UPDATE ON hrm_document_templates
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_document_types IS 'Types of HR documents';
COMMENT ON TABLE hrm_documents IS 'HR documents linked to employees';
COMMENT ON TABLE hrm_document_signatures IS 'Document signature workflow';
COMMENT ON TABLE hrm_document_templates IS 'Templates for generating HR documents';
