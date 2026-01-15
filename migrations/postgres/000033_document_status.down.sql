-- Rollback migration 000033: Drop document_status table

DROP INDEX IF EXISTS idx_document_status_display_order;
DROP TABLE IF EXISTS document_status;
