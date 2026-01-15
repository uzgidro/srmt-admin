-- Migration 000037: Create instructions module (Инструкции)
-- Includes: instruction_type, instructions, file links, document links, status history

-- 1. Create instruction_type table (reference table for instruction types)
CREATE TABLE IF NOT EXISTS instruction_type (
    id SERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert predefined instruction types
INSERT INTO instruction_type (name, description) VALUES
    ('Должностная инструкция', 'Инструкции по должностным обязанностям'),
    ('Рабочая инструкция', 'Инструкции по выполнению рабочих процессов'),
    ('Технологическая инструкция', 'Инструкции по технологическим процессам'),
    ('Инструкция по охране труда', 'Инструкции по технике безопасности'),
    ('Инструкция по пожарной безопасности', 'Инструкции по пожарной безопасности'),
    ('Методическая инструкция', 'Методические указания и рекомендации'),
    ('Операционная инструкция', 'Инструкции по операционным процедурам'),
    ('Инструкция по эксплуатации', 'Инструкции по эксплуатации оборудования'),
    ('Регламент', 'Внутренние регламенты организации'),
    ('Иная инструкция', 'Прочие виды инструкций')
ON CONFLICT (name) DO NOTHING;

-- 2. Create instructions table (main table)
CREATE TABLE IF NOT EXISTS instructions (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    number VARCHAR(100),
    document_date DATE NOT NULL,
    description TEXT,
    type_id INTEGER NOT NULL REFERENCES instruction_type(id) ON DELETE RESTRICT,
    status_id INTEGER NOT NULL REFERENCES document_status(id) ON DELETE RESTRICT DEFAULT 1,
    responsible_contact_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,
    executor_contact_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    due_date DATE,
    parent_document_id BIGINT REFERENCES instructions(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ,
    updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

-- Create trigger for updated_at
CREATE TRIGGER set_timestamp_instructions
    BEFORE UPDATE ON instructions
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- 3. Create junction table for file links
CREATE TABLE IF NOT EXISTS instruction_file_links (
    instruction_id BIGINT NOT NULL REFERENCES instructions(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (instruction_id, file_id)
);

-- 4. Create table for document links (polymorphic links to other documents)
CREATE TABLE IF NOT EXISTS instruction_document_links (
    id BIGSERIAL PRIMARY KEY,
    instruction_id BIGINT NOT NULL REFERENCES instructions(id) ON DELETE CASCADE,
    linked_document_type VARCHAR(50) NOT NULL, -- 'decree', 'report', 'letter', 'instruction', 'legal_document'
    linked_document_id BIGINT NOT NULL,
    link_description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (instruction_id, linked_document_type, linked_document_id)
);

-- 5. Create status history table
CREATE TABLE IF NOT EXISTS instruction_status_history (
    id BIGSERIAL PRIMARY KEY,
    instruction_id BIGINT NOT NULL REFERENCES instructions(id) ON DELETE CASCADE,
    from_status_id INTEGER REFERENCES document_status(id) ON DELETE SET NULL,
    to_status_id INTEGER NOT NULL REFERENCES document_status(id) ON DELETE RESTRICT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    changed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    comment TEXT
);

-- 6. Create trigger function for status history logging
CREATE OR REPLACE FUNCTION log_instruction_status_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only log if status_id actually changed
    IF OLD.status_id IS DISTINCT FROM NEW.status_id THEN
        INSERT INTO instruction_status_history (instruction_id, from_status_id, to_status_id, changed_by_user_id)
        VALUES (NEW.id, OLD.status_id, NEW.status_id, NEW.updated_by_user_id);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for status history
CREATE TRIGGER log_instruction_status_change_trigger
    AFTER UPDATE ON instructions
    FOR EACH ROW
    WHEN (OLD.status_id IS DISTINCT FROM NEW.status_id)
EXECUTE FUNCTION log_instruction_status_change();

-- 7. Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_instructions_type ON instructions(type_id);
CREATE INDEX IF NOT EXISTS idx_instructions_status ON instructions(status_id);
CREATE INDEX IF NOT EXISTS idx_instructions_date ON instructions(document_date DESC);
CREATE INDEX IF NOT EXISTS idx_instructions_due_date ON instructions(due_date);
CREATE INDEX IF NOT EXISTS idx_instructions_name ON instructions(name);
CREATE INDEX IF NOT EXISTS idx_instructions_number ON instructions(number);
CREATE INDEX IF NOT EXISTS idx_instructions_organization ON instructions(organization_id);
CREATE INDEX IF NOT EXISTS idx_instructions_responsible ON instructions(responsible_contact_id);
CREATE INDEX IF NOT EXISTS idx_instructions_executor ON instructions(executor_contact_id);
CREATE INDEX IF NOT EXISTS idx_instructions_parent ON instructions(parent_document_id);
CREATE INDEX IF NOT EXISTS idx_instructions_created_at ON instructions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_instruction_file_links_instruction ON instruction_file_links(instruction_id);
CREATE INDEX IF NOT EXISTS idx_instruction_file_links_file ON instruction_file_links(file_id);

CREATE INDEX IF NOT EXISTS idx_instruction_document_links_instruction ON instruction_document_links(instruction_id);
CREATE INDEX IF NOT EXISTS idx_instruction_document_links_target ON instruction_document_links(linked_document_type, linked_document_id);

CREATE INDEX IF NOT EXISTS idx_instruction_status_history_instruction ON instruction_status_history(instruction_id);
CREATE INDEX IF NOT EXISTS idx_instruction_status_history_changed_at ON instruction_status_history(changed_at DESC);

-- Comments
COMMENT ON TABLE instructions IS 'Инструкции и регламенты';
COMMENT ON TABLE instruction_type IS 'Справочник типов инструкций';
COMMENT ON TABLE instruction_document_links IS 'Связи инструкций с другими документами (полиморфные)';
COMMENT ON TABLE instruction_status_history IS 'История изменений статусов инструкций';
