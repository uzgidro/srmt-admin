-- Migration 000034: Create decrees module (Приказы)
-- Includes: decree_type, decrees, file links, document links, status history

-- 1. Create decree_type table (reference table for decree types)
CREATE TABLE IF NOT EXISTS decree_type (
    id SERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert predefined decree types
INSERT INTO decree_type (name, description) VALUES
    ('Приказ по основной деятельности', 'Приказы, связанные с основной деятельностью организации'),
    ('Приказ по личному составу', 'Приказы о кадровых изменениях'),
    ('Приказ по административно-хозяйственной деятельности', 'Административные и хозяйственные приказы'),
    ('Приказ о командировке', 'Приказы о направлении в командировку'),
    ('Приказ об отпуске', 'Приказы о предоставлении отпуска'),
    ('Приказ о премировании', 'Приказы о премировании сотрудников'),
    ('Приказ о дисциплинарном взыскании', 'Приказы о наложении дисциплинарных взысканий'),
    ('Иной приказ', 'Прочие приказы')
ON CONFLICT (name) DO NOTHING;

-- 2. Create decrees table (main table)
CREATE TABLE IF NOT EXISTS decrees (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    number VARCHAR(100),
    document_date DATE NOT NULL,
    description TEXT,
    type_id INTEGER NOT NULL REFERENCES decree_type(id) ON DELETE RESTRICT,
    status_id INTEGER NOT NULL REFERENCES document_status(id) ON DELETE RESTRICT DEFAULT 1,
    responsible_contact_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,
    executor_contact_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    due_date DATE,
    parent_document_id BIGINT REFERENCES decrees(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ,
    updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

-- Create trigger for updated_at
CREATE TRIGGER set_timestamp_decrees
    BEFORE UPDATE ON decrees
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- 3. Create junction table for file links
CREATE TABLE IF NOT EXISTS decree_file_links (
    decree_id BIGINT NOT NULL REFERENCES decrees(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (decree_id, file_id)
);

-- 4. Create table for document links (polymorphic links to other documents)
CREATE TABLE IF NOT EXISTS decree_document_links (
    id BIGSERIAL PRIMARY KEY,
    decree_id BIGINT NOT NULL REFERENCES decrees(id) ON DELETE CASCADE,
    linked_document_type VARCHAR(50) NOT NULL, -- 'decree', 'report', 'letter', 'instruction', 'legal_document'
    linked_document_id BIGINT NOT NULL,
    link_description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (decree_id, linked_document_type, linked_document_id)
);

-- 5. Create status history table
CREATE TABLE IF NOT EXISTS decree_status_history (
    id BIGSERIAL PRIMARY KEY,
    decree_id BIGINT NOT NULL REFERENCES decrees(id) ON DELETE CASCADE,
    from_status_id INTEGER REFERENCES document_status(id) ON DELETE SET NULL,
    to_status_id INTEGER NOT NULL REFERENCES document_status(id) ON DELETE RESTRICT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    changed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    comment TEXT
);

-- 6. Create trigger function for status history logging
CREATE OR REPLACE FUNCTION log_decree_status_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only log if status_id actually changed
    IF OLD.status_id IS DISTINCT FROM NEW.status_id THEN
        INSERT INTO decree_status_history (decree_id, from_status_id, to_status_id, changed_by_user_id)
        VALUES (NEW.id, OLD.status_id, NEW.status_id, NEW.updated_by_user_id);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for status history
CREATE TRIGGER log_decree_status_change_trigger
    AFTER UPDATE ON decrees
    FOR EACH ROW
    WHEN (OLD.status_id IS DISTINCT FROM NEW.status_id)
EXECUTE FUNCTION log_decree_status_change();

-- 7. Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_decrees_type ON decrees(type_id);
CREATE INDEX IF NOT EXISTS idx_decrees_status ON decrees(status_id);
CREATE INDEX IF NOT EXISTS idx_decrees_date ON decrees(document_date DESC);
CREATE INDEX IF NOT EXISTS idx_decrees_due_date ON decrees(due_date);
CREATE INDEX IF NOT EXISTS idx_decrees_name ON decrees(name);
CREATE INDEX IF NOT EXISTS idx_decrees_number ON decrees(number);
CREATE INDEX IF NOT EXISTS idx_decrees_organization ON decrees(organization_id);
CREATE INDEX IF NOT EXISTS idx_decrees_responsible ON decrees(responsible_contact_id);
CREATE INDEX IF NOT EXISTS idx_decrees_executor ON decrees(executor_contact_id);
CREATE INDEX IF NOT EXISTS idx_decrees_parent ON decrees(parent_document_id);
CREATE INDEX IF NOT EXISTS idx_decrees_created_at ON decrees(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_decree_file_links_decree ON decree_file_links(decree_id);
CREATE INDEX IF NOT EXISTS idx_decree_file_links_file ON decree_file_links(file_id);

CREATE INDEX IF NOT EXISTS idx_decree_document_links_decree ON decree_document_links(decree_id);
CREATE INDEX IF NOT EXISTS idx_decree_document_links_target ON decree_document_links(linked_document_type, linked_document_id);

CREATE INDEX IF NOT EXISTS idx_decree_status_history_decree ON decree_status_history(decree_id);
CREATE INDEX IF NOT EXISTS idx_decree_status_history_changed_at ON decree_status_history(changed_at DESC);

-- Comments
COMMENT ON TABLE decrees IS 'Приказы организации';
COMMENT ON TABLE decree_type IS 'Справочник типов приказов';
COMMENT ON TABLE decree_document_links IS 'Связи приказов с другими документами (полиморфные)';
COMMENT ON TABLE decree_status_history IS 'История изменений статусов приказов';
