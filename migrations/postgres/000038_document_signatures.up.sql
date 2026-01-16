-- Migration: 000038_document_signatures
-- Description: Add document signing functionality with resolutions

-- 1. Add new statuses to document_status table
INSERT INTO document_status (code, name, description, display_order, is_terminal) VALUES
    ('pending_signature', 'На подписании', 'Документ ожидает подписи руководителя', 8, FALSE),
    ('signed', 'Подписан', 'Документ подписан руководителем', 9, FALSE),
    ('signature_rejected', 'Отклонён при подписании', 'Документ отклонён при подписании', 10, TRUE)
ON CONFLICT (code) DO NOTHING;

-- 2. Create document_signatures table (polymorphic for all document types)
CREATE TABLE IF NOT EXISTS document_signatures (
    id BIGSERIAL PRIMARY KEY,

    -- Polymorphic reference to document
    document_type VARCHAR(50) NOT NULL,     -- 'decree', 'report', 'letter', 'instruction'
    document_id BIGINT NOT NULL,

    -- Signature action result
    action VARCHAR(20) NOT NULL,            -- 'signed', 'rejected'
    resolution_text TEXT,                   -- Resolution text (for signed)
    rejection_reason TEXT,                  -- Rejection reason (for rejected)

    -- Optional: assign executor and due date with resolution
    assigned_executor_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    assigned_due_date DATE,

    -- Who signed and when
    signed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    signed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_signature_action CHECK (action IN ('signed', 'rejected')),
    CONSTRAINT chk_document_type CHECK (document_type IN ('decree', 'report', 'letter', 'instruction'))
);

-- 3. Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_doc_signatures_type_id ON document_signatures(document_type, document_id);
CREATE INDEX IF NOT EXISTS idx_doc_signatures_action ON document_signatures(action);
CREATE INDEX IF NOT EXISTS idx_doc_signatures_signed_by ON document_signatures(signed_by_user_id);
CREATE INDEX IF NOT EXISTS idx_doc_signatures_signed_at ON document_signatures(signed_at DESC);

-- 4. Add comments
COMMENT ON TABLE document_signatures IS 'Document signatures and resolutions for all document types';
COMMENT ON COLUMN document_signatures.document_type IS 'Type of document: decree, report, letter, instruction';
COMMENT ON COLUMN document_signatures.action IS 'Signature action: signed or rejected';
COMMENT ON COLUMN document_signatures.resolution_text IS 'Resolution text when document is signed';
COMMENT ON COLUMN document_signatures.rejection_reason IS 'Reason for rejection when document is rejected';
COMMENT ON COLUMN document_signatures.assigned_executor_id IS 'Optional executor assigned with resolution';
COMMENT ON COLUMN document_signatures.assigned_due_date IS 'Optional due date assigned with resolution';
