-- Migration 000033: Create document_status table (shared workflow statuses for all document types)
-- This table provides a common status workflow for decrees, reports, letters, and instructions

CREATE TABLE IF NOT EXISTS document_status (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_terminal BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert predefined workflow statuses
INSERT INTO document_status (code, name, description, display_order, is_terminal) VALUES
    ('draft', 'Черновик', 'Документ создан, но ещё не отправлен на согласование', 1, FALSE),
    ('pending_approval', 'На согласовании', 'Документ отправлен на согласование и ожидает решения', 2, FALSE),
    ('approved', 'Утверждён', 'Документ согласован и утверждён', 3, FALSE),
    ('rejected', 'Отклонён', 'Документ отклонён при согласовании', 4, TRUE),
    ('in_execution', 'На исполнении', 'Документ принят к исполнению', 5, FALSE),
    ('executed', 'Исполнен', 'Документ полностью исполнен', 6, TRUE),
    ('cancelled', 'Отменён', 'Документ отменён', 7, TRUE)
ON CONFLICT (code) DO NOTHING;

-- Create index for ordering
CREATE INDEX IF NOT EXISTS idx_document_status_display_order ON document_status(display_order);

COMMENT ON TABLE document_status IS 'Общий справочник статусов workflow для всех типов документов';
COMMENT ON COLUMN document_status.code IS 'Уникальный код статуса для использования в логике приложения';
COMMENT ON COLUMN document_status.is_terminal IS 'Флаг конечного состояния (из него нельзя перейти в другое)';
