-- Migration 000037 down: Drop instructions module

-- Drop triggers
DROP TRIGGER IF EXISTS log_instruction_status_change_trigger ON instructions;
DROP TRIGGER IF EXISTS set_timestamp_instructions ON instructions;

-- Drop trigger function
DROP FUNCTION IF EXISTS log_instruction_status_change();

-- Drop tables in reverse order
DROP TABLE IF EXISTS instruction_status_history;
DROP TABLE IF EXISTS instruction_document_links;
DROP TABLE IF EXISTS instruction_file_links;
DROP TABLE IF EXISTS instructions;
DROP TABLE IF EXISTS instruction_type;
