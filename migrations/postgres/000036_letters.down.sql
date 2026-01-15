-- Migration 000036 down: Drop letters module

-- Drop triggers
DROP TRIGGER IF EXISTS log_letter_status_change_trigger ON letters;
DROP TRIGGER IF EXISTS set_timestamp_letters ON letters;

-- Drop trigger function
DROP FUNCTION IF EXISTS log_letter_status_change();

-- Drop tables in reverse order
DROP TABLE IF EXISTS letter_status_history;
DROP TABLE IF EXISTS letter_document_links;
DROP TABLE IF EXISTS letter_file_links;
DROP TABLE IF EXISTS letters;
DROP TABLE IF EXISTS letter_type;
