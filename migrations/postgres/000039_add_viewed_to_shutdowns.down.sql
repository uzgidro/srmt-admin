DROP INDEX IF EXISTS idx_shutdowns_viewed;
ALTER TABLE shutdowns DROP COLUMN IF EXISTS viewed;
