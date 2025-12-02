-- Drop fast_calls table and its dependencies
DROP INDEX IF EXISTS idx_fast_calls_position;
DROP INDEX IF EXISTS idx_fast_calls_contact_id;
DROP TABLE IF EXISTS fast_calls;
