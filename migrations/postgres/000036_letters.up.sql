-- Migration 000036: Create letters module (Письма)
-- Includes: letter_type, letters, file links, document links, status history

-- 1. Create letter_type table (reference table for letter types)
CREATE TABLE IF NOT EXISTS letter_type (
    id SERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert predefined letter types
INSERT INTO letter_type (name, description) VALUES
    ('Входящее письмо', 'Письма, полученные от внешних организаций'),
    ('Исходящее письмо', 'Письма, отправляемые во внешние организации'),
    ('Внутреннее письмо', 'Письма между подразделениями организации'),
    ('Запрос информации', 'Письма-запросы информации'),
    ('Ответ на запрос', 'Письма-ответы на запросы'),
    ('Уведомление', 'Письма-уведомления'),
    ('Претензия', 'Письма-претензии'),
    ('Благодарственное письмо', 'Благодарственные письма'),
    ('Гарантийное письмо', 'Гарантийные письма'),
    ('Сопроводительное письмо', 'Сопроводительные письма к документам'),
    ('Иное письмо', 'Прочие виды писем')
ON CONFLICT (name) DO NOTHING;

-- 2. Create letters table (main table)
CREATE TABLE IF NOT EXISTS letters (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    number VARCHAR(100),
    document_date DATE NOT NULL,
    description TEXT,
    type_id INTEGER NOT NULL REFERENCES letter_type(id) ON DELETE RESTRICT,
    status_id INTEGER NOT NULL REFERENCES document_status(id) ON DELETE RESTRICT DEFAULT 1,
    responsible_contact_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,
    executor_contact_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    due_date DATE,
    parent_document_id BIGINT REFERENCES letters(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ,
    updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

-- Create trigger for updated_at
CREATE TRIGGER set_timestamp_letters
    BEFORE UPDATE ON letters
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- 3. Create junction table for file links
CREATE TABLE IF NOT EXISTS letter_file_links (
    letter_id BIGINT NOT NULL REFERENCES letters(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (letter_id, file_id)
);

-- 4. Create table for document links (polymorphic links to other documents)
CREATE TABLE IF NOT EXISTS letter_document_links (
    id BIGSERIAL PRIMARY KEY,
    letter_id BIGINT NOT NULL REFERENCES letters(id) ON DELETE CASCADE,
    linked_document_type VARCHAR(50) NOT NULL, -- 'decree', 'report', 'letter', 'instruction', 'legal_document'
    linked_document_id BIGINT NOT NULL,
    link_description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (letter_id, linked_document_type, linked_document_id)
);

-- 5. Create status history table
CREATE TABLE IF NOT EXISTS letter_status_history (
    id BIGSERIAL PRIMARY KEY,
    letter_id BIGINT NOT NULL REFERENCES letters(id) ON DELETE CASCADE,
    from_status_id INTEGER REFERENCES document_status(id) ON DELETE SET NULL,
    to_status_id INTEGER NOT NULL REFERENCES document_status(id) ON DELETE RESTRICT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    changed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    comment TEXT
);

-- 6. Create trigger function for status history logging
CREATE OR REPLACE FUNCTION log_letter_status_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only log if status_id actually changed
    IF OLD.status_id IS DISTINCT FROM NEW.status_id THEN
        INSERT INTO letter_status_history (letter_id, from_status_id, to_status_id, changed_by_user_id)
        VALUES (NEW.id, OLD.status_id, NEW.status_id, NEW.updated_by_user_id);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for status history
CREATE TRIGGER log_letter_status_change_trigger
    AFTER UPDATE ON letters
    FOR EACH ROW
    WHEN (OLD.status_id IS DISTINCT FROM NEW.status_id)
EXECUTE FUNCTION log_letter_status_change();

-- 7. Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_letters_type ON letters(type_id);
CREATE INDEX IF NOT EXISTS idx_letters_status ON letters(status_id);
CREATE INDEX IF NOT EXISTS idx_letters_date ON letters(document_date DESC);
CREATE INDEX IF NOT EXISTS idx_letters_due_date ON letters(due_date);
CREATE INDEX IF NOT EXISTS idx_letters_name ON letters(name);
CREATE INDEX IF NOT EXISTS idx_letters_number ON letters(number);
CREATE INDEX IF NOT EXISTS idx_letters_organization ON letters(organization_id);
CREATE INDEX IF NOT EXISTS idx_letters_responsible ON letters(responsible_contact_id);
CREATE INDEX IF NOT EXISTS idx_letters_executor ON letters(executor_contact_id);
CREATE INDEX IF NOT EXISTS idx_letters_parent ON letters(parent_document_id);
CREATE INDEX IF NOT EXISTS idx_letters_created_at ON letters(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_letter_file_links_letter ON letter_file_links(letter_id);
CREATE INDEX IF NOT EXISTS idx_letter_file_links_file ON letter_file_links(file_id);

CREATE INDEX IF NOT EXISTS idx_letter_document_links_letter ON letter_document_links(letter_id);
CREATE INDEX IF NOT EXISTS idx_letter_document_links_target ON letter_document_links(linked_document_type, linked_document_id);

CREATE INDEX IF NOT EXISTS idx_letter_status_history_letter ON letter_status_history(letter_id);
CREATE INDEX IF NOT EXISTS idx_letter_status_history_changed_at ON letter_status_history(changed_at DESC);

-- Comments
COMMENT ON TABLE letters IS 'Письма (входящие, исходящие, внутренние)';
COMMENT ON TABLE letter_type IS 'Справочник типов писем';
COMMENT ON TABLE letter_document_links IS 'Связи писем с другими документами (полиморфные)';
COMMENT ON TABLE letter_status_history IS 'История изменений статусов писем';
