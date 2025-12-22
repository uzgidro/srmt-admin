-- Drop indexes
DROP INDEX IF EXISTS idx_files_category_target_date;
DROP INDEX IF EXISTS idx_files_target_date;

-- Drop target_date column from files table
ALTER TABLE files DROP COLUMN IF EXISTS target_date;
