-- Drop receptions table and its dependencies
DROP TRIGGER IF EXISTS set_timestamp_receptions ON receptions;
DROP INDEX IF EXISTS idx_receptions_visitor;
DROP INDEX IF EXISTS idx_receptions_created_by;
DROP INDEX IF EXISTS idx_receptions_created_at;
DROP INDEX IF EXISTS idx_receptions_status;
DROP INDEX IF EXISTS idx_receptions_date;
DROP TABLE IF EXISTS receptions;
