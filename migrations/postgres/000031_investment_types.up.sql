-- Migration 000031: Add investment types and extend status system
-- This migration adds support for multiple financing types with type-specific statuses

-- Step 1: Create investment_type table
CREATE TABLE IF NOT EXISTS investment_type (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert the 3 financing types
INSERT INTO investment_type (id, name, description) VALUES
    (1, 'Перспективные проекты за счёт собственные средства', 'Инвестиции за счет собственных средств'),
    (2, 'Перспективные проекты за счет частных инвестиций (ГЧП)', 'Инвестиции за счет частных инвестиций (государственно-частное партнерство)'),
    (3, 'Перспективные проекты за счет кредитов под государственной гарантией', 'Инвестиции за счет кредитов под государственной гарантией')
ON CONFLICT (name) DO NOTHING;

-- Step 2: Extend investment_status table
ALTER TABLE investment_status ADD COLUMN IF NOT EXISTS type_id INTEGER;
ALTER TABLE investment_status ADD COLUMN IF NOT EXISTS display_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE investment_status ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Step 3: Add foreign key constraint
ALTER TABLE investment_status ADD CONSTRAINT fk_investment_status_type
    FOREIGN KEY (type_id) REFERENCES investment_type(id) ON DELETE RESTRICT;

-- Step 4: Update unique constraint (allow same name for different types)
ALTER TABLE investment_status DROP CONSTRAINT IF EXISTS investment_status_name_key;
CREATE UNIQUE INDEX idx_investment_status_unique_name_type
    ON investment_status(name, COALESCE(type_id, 0));

-- Step 5: Create indexes
CREATE INDEX IF NOT EXISTS idx_investment_status_type ON investment_status(type_id);
CREATE INDEX IF NOT EXISTS idx_investment_status_display_order ON investment_status(display_order);

-- Step 6: Insert new statuses (without deleting old ones yet)
-- Insert shared statuses (type_id = NULL means available for all types)
INSERT INTO investment_status (name, description, type_id, display_order) VALUES
    ('Разработка ТЗ', 'Разработка технического задания', NULL, 1),
    ('Экспертиза ТЗ', 'Экспертиза технического задания', NULL, 2),
    ('Разработка ТЭО', 'Разработка технико-экономического обоснования', NULL, 3),
    ('Экспертиза ТЭО', 'Экспертиза технико-экономического обоснования', NULL, 4),
    ('Утверждение ТЭО', 'Утверждение технико-экономического обоснования', NULL, 5)
ON CONFLICT (name, COALESCE(type_id, 0)) DO NOTHING;

-- Own Funds specific statuses (type_id = 1)
INSERT INTO investment_status (name, description, type_id, display_order) VALUES
    ('Переход на активную фазу', 'Инвестиция переведена в активную фазу реализации', 1, 6)
ON CONFLICT (name, COALESCE(type_id, 0)) DO NOTHING;

-- PPP specific statuses (type_id = 2)
INSERT INTO investment_status (name, description, type_id, display_order) VALUES
    ('Согласование тарифа', 'Согласование тарифной политики для ГЧП проекта', 2, 6),
    ('Утверждение тарифа', 'Утверждение тарифной политики', 2, 7),
    ('Подписание договоров по поставке электроэнергии (PCA) и государственной поддержки (GSA)',
     'Подписание договоров PCA и GSA', 2, 8),
    ('Учреждение совместной инвестиционной компании',
     'Учреждение совместной инвестиционной компании', 2, 9),
    ('Переход на активную фазу', 'ГЧП проект переведен в активную фазу', 2, 10)
ON CONFLICT (name, COALESCE(type_id, 0)) DO NOTHING;

-- State Guarantee Loans specific statuses (type_id = 3)
INSERT INTO investment_status (name, description, type_id, display_order) VALUES
    ('Рассмотрение проектных документов иностранным банком',
     'Рассмотрение проектных документов иностранным банком', 3, 6),
    ('Получение одобрения банка по финансированию проекта',
     'Получение одобрения банка по финансированию проекта', 3, 7),
    ('Проведение тендерных торгов для определения подрядчика',
     'Проведение тендерных торгов для определения подрядчика', 3, 8),
    ('Согласование тендерного соглашения с банком',
     'Согласование тендерного соглашения с банком', 3, 9),
    ('Подписание заёмного соглашения',
     'Подписание заёмного соглашения', 3, 10),
    ('Получение государственной гарантии на заёмное соглашение',
     'Получение государственной гарантии на заёмное соглашение', 3, 11),
    ('Получение юридического заключения на заёмное соглашение',
     'Получение юридического заключения на заёмное соглашение', 3, 12),
    ('Рассмотрение банком пакета документов',
     'Рассмотрение банком пакета документов', 3, 13),
    ('Одобрение кредита и открытие финансирования',
     'Одобрение кредита и открытие финансирования', 3, 14)
ON CONFLICT (name, COALESCE(type_id, 0)) DO NOTHING;

-- Step 7: Add type_id column to investments table
ALTER TABLE investments ADD COLUMN IF NOT EXISTS type_id INTEGER;

-- Step 8: Set default type (Own Funds = 1) for existing records
UPDATE investments SET type_id = 1 WHERE type_id IS NULL;

-- Step 9: Update existing investments' status_id to match the new status IDs
-- Set to first shared status for existing investments
UPDATE investments
SET status_id = (SELECT id FROM investment_status WHERE name = 'Разработка ТЗ' AND type_id IS NULL LIMIT 1)
WHERE type_id = 1;

-- Step 9a: Delete old statuses (now safe after existing investments have been updated)
-- Only delete statuses that are NOT in our new system
DELETE FROM investment_status
WHERE name IN ('Planned', 'In Progress', 'Completed', 'Cancelled')
  AND type_id IS NULL
  AND description IN (
    'Investment is planned but not yet started',
    'Investment activities are currently underway',
    'Investment has been successfully completed',
    'Investment was cancelled or abandoned'
  );

-- Step 10: Make type_id NOT NULL and add constraints
ALTER TABLE investments ALTER COLUMN type_id SET NOT NULL;
ALTER TABLE investments ADD CONSTRAINT fk_investments_type
    FOREIGN KEY (type_id) REFERENCES investment_type(id) ON DELETE RESTRICT;

-- Step 11: Create index for type filtering
CREATE INDEX IF NOT EXISTS idx_investments_type ON investments(type_id);

-- Step 12: Create validation function and trigger
CREATE OR REPLACE FUNCTION validate_investment_status_type()
RETURNS TRIGGER AS $$
BEGIN
    -- Check if the status is valid for this investment's type
    -- Status is valid if:
    -- 1. Status has no type_id (shared status), OR
    -- 2. Status type_id matches investment type_id
    IF NOT EXISTS (
        SELECT 1 FROM investment_status
        WHERE id = NEW.status_id
        AND (type_id IS NULL OR type_id = NEW.type_id)
    ) THEN
        RAISE EXCEPTION 'Status ID % is not valid for investment type ID %',
            NEW.status_id, NEW.type_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER validate_investment_status_before_insert_update
    BEFORE INSERT OR UPDATE ON investments
    FOR EACH ROW
    EXECUTE FUNCTION validate_investment_status_type();
