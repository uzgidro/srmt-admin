-- Migration: 000038_document_signatures (DOWN)
-- Description: Remove document signing functionality

-- 1. Drop table
DROP TABLE IF EXISTS document_signatures;

-- 2. Remove statuses (be careful - only remove if no documents use them)
DELETE FROM document_status WHERE code IN ('pending_signature', 'signed', 'signature_rejected');
