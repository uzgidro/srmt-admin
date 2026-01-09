-- Drop migration 000031: Remove investment types and revert to simple status system

-- Drop trigger and function
DROP TRIGGER IF EXISTS validate_investment_status_before_insert_update ON investments;
DROP FUNCTION IF EXISTS validate_investment_status_type();

-- Remove type_id from investments
ALTER TABLE investments DROP CONSTRAINT IF EXISTS fk_investments_type;
DROP INDEX IF EXISTS idx_investments_type;
ALTER TABLE investments DROP COLUMN IF EXISTS type_id;

-- Revert investment_status table
DROP INDEX IF EXISTS idx_investment_status_unique_name_type;
DROP INDEX IF EXISTS idx_investment_status_type;
DROP INDEX IF EXISTS idx_investment_status_display_order;

ALTER TABLE investment_status DROP CONSTRAINT IF EXISTS fk_investment_status_type;
ALTER TABLE investment_status DROP COLUMN IF EXISTS type_id;
ALTER TABLE investment_status DROP COLUMN IF EXISTS display_order;
ALTER TABLE investment_status DROP COLUMN IF EXISTS created_at;

-- Restore original unique constraint
ALTER TABLE investment_status ADD CONSTRAINT investment_status_name_key UNIQUE (name);

-- Restore original statuses
DELETE FROM investment_status;
INSERT INTO investment_status (name, description) VALUES
    ('Planned', 'Investment is planned but not yet started'),
    ('In Progress', 'Investment activities are currently underway'),
    ('Completed', 'Investment has been successfully completed'),
    ('Cancelled', 'Investment was cancelled or abandoned')
ON CONFLICT (name) DO NOTHING;

-- Drop investment_type table
DROP TABLE IF EXISTS investment_type;
