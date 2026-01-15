-- Migration 000035 down: Drop reports module

-- Drop triggers
DROP TRIGGER IF EXISTS log_report_status_change_trigger ON reports;
DROP TRIGGER IF EXISTS set_timestamp_reports ON reports;

-- Drop trigger function
DROP FUNCTION IF EXISTS log_report_status_change();

-- Drop tables in reverse order
DROP TABLE IF EXISTS report_status_history;
DROP TABLE IF EXISTS report_document_links;
DROP TABLE IF EXISTS report_file_links;
DROP TABLE IF EXISTS reports;
DROP TABLE IF EXISTS report_type;
