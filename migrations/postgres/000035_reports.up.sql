-- Migration 000035: Create reports module (Рапорты)
-- Includes: report_type, reports, file links, document links, status history

-- 1. Create report_type table (reference table for report types)
CREATE TABLE IF NOT EXISTS report_type (
    id SERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert predefined report types
INSERT INTO report_type (name, description) VALUES
    ('Рапорт о проделанной работе', 'Отчёт о выполненных задачах за период'),
    ('Рапорт о происшествии', 'Доклад о чрезвычайной ситуации или инциденте'),
    ('Рапорт о командировке', 'Отчёт по результатам командировки'),
    ('Рапорт о нарушении', 'Доклад о выявленных нарушениях'),
    ('Рапорт о состоянии объекта', 'Отчёт о техническом состоянии объекта'),
    ('Рапорт о выполнении поручения', 'Отчёт об исполнении поручения руководства'),
    ('Докладная записка', 'Докладная записка руководству'),
    ('Служебная записка', 'Служебная записка внутреннего характера'),
    ('Объяснительная записка', 'Объяснительная записка'),
    ('Иной рапорт', 'Прочие рапорты и докладные')
ON CONFLICT (name) DO NOTHING;

-- 2. Create reports table (main table)
CREATE TABLE IF NOT EXISTS reports (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    number VARCHAR(100),
    document_date DATE NOT NULL,
    description TEXT,
    type_id INTEGER NOT NULL REFERENCES report_type(id) ON DELETE RESTRICT,
    status_id INTEGER NOT NULL REFERENCES document_status(id) ON DELETE RESTRICT DEFAULT 1,
    responsible_contact_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL,
    executor_contact_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,
    due_date DATE,
    parent_document_id BIGINT REFERENCES reports(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ,
    updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

-- Create trigger for updated_at
CREATE TRIGGER set_timestamp_reports
    BEFORE UPDATE ON reports
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- 3. Create junction table for file links
CREATE TABLE IF NOT EXISTS report_file_links (
    report_id BIGINT NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (report_id, file_id)
);

-- 4. Create table for document links (polymorphic links to other documents)
CREATE TABLE IF NOT EXISTS report_document_links (
    id BIGSERIAL PRIMARY KEY,
    report_id BIGINT NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    linked_document_type VARCHAR(50) NOT NULL, -- 'decree', 'report', 'letter', 'instruction', 'legal_document'
    linked_document_id BIGINT NOT NULL,
    link_description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE (report_id, linked_document_type, linked_document_id)
);

-- 5. Create status history table
CREATE TABLE IF NOT EXISTS report_status_history (
    id BIGSERIAL PRIMARY KEY,
    report_id BIGINT NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    from_status_id INTEGER REFERENCES document_status(id) ON DELETE SET NULL,
    to_status_id INTEGER NOT NULL REFERENCES document_status(id) ON DELETE RESTRICT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    changed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    comment TEXT
);

-- 6. Create trigger function for status history logging
CREATE OR REPLACE FUNCTION log_report_status_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only log if status_id actually changed
    IF OLD.status_id IS DISTINCT FROM NEW.status_id THEN
        INSERT INTO report_status_history (report_id, from_status_id, to_status_id, changed_by_user_id)
        VALUES (NEW.id, OLD.status_id, NEW.status_id, NEW.updated_by_user_id);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for status history
CREATE TRIGGER log_report_status_change_trigger
    AFTER UPDATE ON reports
    FOR EACH ROW
    WHEN (OLD.status_id IS DISTINCT FROM NEW.status_id)
EXECUTE FUNCTION log_report_status_change();

-- 7. Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_reports_type ON reports(type_id);
CREATE INDEX IF NOT EXISTS idx_reports_status ON reports(status_id);
CREATE INDEX IF NOT EXISTS idx_reports_date ON reports(document_date DESC);
CREATE INDEX IF NOT EXISTS idx_reports_due_date ON reports(due_date);
CREATE INDEX IF NOT EXISTS idx_reports_name ON reports(name);
CREATE INDEX IF NOT EXISTS idx_reports_number ON reports(number);
CREATE INDEX IF NOT EXISTS idx_reports_organization ON reports(organization_id);
CREATE INDEX IF NOT EXISTS idx_reports_responsible ON reports(responsible_contact_id);
CREATE INDEX IF NOT EXISTS idx_reports_executor ON reports(executor_contact_id);
CREATE INDEX IF NOT EXISTS idx_reports_parent ON reports(parent_document_id);
CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_report_file_links_report ON report_file_links(report_id);
CREATE INDEX IF NOT EXISTS idx_report_file_links_file ON report_file_links(file_id);

CREATE INDEX IF NOT EXISTS idx_report_document_links_report ON report_document_links(report_id);
CREATE INDEX IF NOT EXISTS idx_report_document_links_target ON report_document_links(linked_document_type, linked_document_id);

CREATE INDEX IF NOT EXISTS idx_report_status_history_report ON report_status_history(report_id);
CREATE INDEX IF NOT EXISTS idx_report_status_history_changed_at ON report_status_history(changed_at DESC);

-- Comments
COMMENT ON TABLE reports IS 'Рапорты и докладные записки';
COMMENT ON TABLE report_type IS 'Справочник типов рапортов';
COMMENT ON TABLE report_document_links IS 'Связи рапортов с другими документами (полиморфные)';
COMMENT ON TABLE report_status_history IS 'История изменений статусов рапортов';
