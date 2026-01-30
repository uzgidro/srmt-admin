-- HRM Personnel Records (Documents and Transfers)

-- Personnel documents (passport, diplomas, certificates, etc.)
CREATE TABLE IF NOT EXISTS hrm_personnel_documents (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    document_type VARCHAR(100) NOT NULL, -- passport, diploma, certificate, contract, military_id, etc.
    document_number VARCHAR(100),
    document_series VARCHAR(50),

    issued_by VARCHAR(255),
    issued_date DATE,
    expiry_date DATE,

    -- File attachment
    file_id BIGINT REFERENCES files(id) ON DELETE SET NULL,

    notes TEXT,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    verified_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_personnel_documents_employee ON hrm_personnel_documents(employee_id);
CREATE INDEX idx_hrm_personnel_documents_type ON hrm_personnel_documents(document_type);
CREATE INDEX idx_hrm_personnel_documents_expiry ON hrm_personnel_documents(expiry_date);

CREATE TRIGGER set_timestamp_hrm_personnel_documents
    BEFORE UPDATE ON hrm_personnel_documents
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

-- Transfer history (position/department changes)
CREATE TABLE IF NOT EXISTS hrm_transfers (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES hrm_employees(id) ON DELETE CASCADE,

    -- From (previous position)
    from_department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    from_position_id BIGINT REFERENCES positions(id) ON DELETE SET NULL,
    from_organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,

    -- To (new position)
    to_department_id BIGINT REFERENCES departments(id) ON DELETE SET NULL,
    to_position_id BIGINT REFERENCES positions(id) ON DELETE SET NULL,
    to_organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,

    transfer_type VARCHAR(50) NOT NULL, -- promotion, demotion, lateral, relocation
    transfer_reason TEXT,
    effective_date DATE NOT NULL,

    -- Approval
    order_number VARCHAR(100),
    order_date DATE,
    order_file_id BIGINT REFERENCES files(id) ON DELETE SET NULL,

    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_hrm_transfers_employee ON hrm_transfers(employee_id);
CREATE INDEX idx_hrm_transfers_effective_date ON hrm_transfers(effective_date);
CREATE INDEX idx_hrm_transfers_type ON hrm_transfers(transfer_type);

CREATE TRIGGER set_timestamp_hrm_transfers
    BEFORE UPDATE ON hrm_transfers
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_timestamp();

COMMENT ON TABLE hrm_personnel_documents IS 'Employee personal documents (passport, diplomas, etc.)';
COMMENT ON TABLE hrm_transfers IS 'Employee position/department transfer history';
