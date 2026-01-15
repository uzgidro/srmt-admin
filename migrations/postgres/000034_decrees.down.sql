-- Rollback migration 000034: Drop decrees module

-- Drop triggers first
DROP TRIGGER IF EXISTS log_decree_status_change_trigger ON decrees;
DROP TRIGGER IF EXISTS set_timestamp_decrees ON decrees;

-- Drop function
DROP FUNCTION IF EXISTS log_decree_status_change();

-- Drop indexes
DROP INDEX IF EXISTS idx_decree_status_history_changed_at;
DROP INDEX IF EXISTS idx_decree_status_history_decree;
DROP INDEX IF EXISTS idx_decree_document_links_target;
DROP INDEX IF EXISTS idx_decree_document_links_decree;
DROP INDEX IF EXISTS idx_decree_file_links_file;
DROP INDEX IF EXISTS idx_decree_file_links_decree;
DROP INDEX IF EXISTS idx_decrees_created_at;
DROP INDEX IF EXISTS idx_decrees_parent;
DROP INDEX IF EXISTS idx_decrees_executor;
DROP INDEX IF EXISTS idx_decrees_responsible;
DROP INDEX IF EXISTS idx_decrees_organization;
DROP INDEX IF EXISTS idx_decrees_number;
DROP INDEX IF EXISTS idx_decrees_name;
DROP INDEX IF EXISTS idx_decrees_due_date;
DROP INDEX IF EXISTS idx_decrees_date;
DROP INDEX IF EXISTS idx_decrees_status;
DROP INDEX IF EXISTS idx_decrees_type;

-- Drop tables in correct order (dependencies)
DROP TABLE IF EXISTS decree_status_history;
DROP TABLE IF EXISTS decree_document_links;
DROP TABLE IF EXISTS decree_file_links;
DROP TABLE IF EXISTS decrees;
DROP TABLE IF EXISTS decree_type;
