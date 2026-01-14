-- Create legal_document_type table (reference table for document types)
CREATE TABLE IF NOT EXISTS legal_document_type (
    id SERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL UNIQUE,
    description TEXT
);

-- Insert predefined document types
INSERT INTO legal_document_type (name, description) VALUES
    ('Перечни Законов и иных нормативных актов, принятых Верховным Советом и Олий Мажлисом Республики Узбекистан', 'Законы и нормативные акты'),
    ('Перечни Указов Президента Республики Узбекистан', 'Указы Президента'),
    ('Перечни Постановлений Президента Республики Узбекистан', 'Постановления Президента'),
    ('Перечни Распоряжений Президента Республики Узбекистан', 'Распоряжения Президента'),
    ('Перечни Постановлений Правительства Республики Узбекистан', 'Постановления Правительства'),
    ('Перечни Распоряжений Кабинета Министров Республики Узбекистан', 'Распоряжения Кабинета Министров'),
    ('Перечни ведомственных нормативных актов министерств, государственных комитетов и ведомств', 'Ведомственные нормативные акты'),
    ('Перечни документов, опубликованных в "Собрании законодательства Республики Узбекистан"', 'Собрание законодательства'),
    ('Иные перечни', 'Прочие документы'),
    ('Приказы Узбекгидроэнерго', 'Приказы компании'),
    ('Протоколы Узбекгидроэнерго', 'Протоколы компании')
ON CONFLICT (name) DO NOTHING;

-- Create legal_documents table (main documents table)
CREATE TABLE IF NOT EXISTS legal_documents (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(500) NOT NULL,
    number VARCHAR(100),
    document_date DATE NOT NULL,
    type_id INTEGER NOT NULL REFERENCES legal_document_type(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ,
    updated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL
);

-- Create trigger for updated_at (trigger function already exists from migration 000010)
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON legal_documents
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Create junction table for many-to-many file relationship
CREATE TABLE IF NOT EXISTS legal_document_file_links (
    document_id BIGINT NOT NULL REFERENCES legal_documents(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (document_id, file_id)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_legal_documents_type ON legal_documents(type_id);
CREATE INDEX IF NOT EXISTS idx_legal_documents_date ON legal_documents(document_date DESC);
CREATE INDEX IF NOT EXISTS idx_legal_documents_name ON legal_documents(name);
CREATE INDEX IF NOT EXISTS idx_legal_documents_number ON legal_documents(number);
CREATE INDEX IF NOT EXISTS idx_legal_documents_created_at ON legal_documents(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_legal_document_file_links_document ON legal_document_file_links(document_id);
CREATE INDEX IF NOT EXISTS idx_legal_document_file_links_file ON legal_document_file_links(file_id);
