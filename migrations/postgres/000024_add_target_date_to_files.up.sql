-- Add target_date column to files table (date only, without time)
ALTER TABLE files ADD COLUMN target_date DATE NOT NULL DEFAULT CURRENT_DATE;

-- Set target_date for existing files to their created_at date
UPDATE files SET target_date = created_at::DATE;

-- Create index for faster queries on target_date
CREATE INDEX IF NOT EXISTS idx_files_target_date ON files (target_date);

-- Create composite index for category and target_date lookups
CREATE INDEX IF NOT EXISTS idx_files_category_target_date ON files (category_id, target_date DESC);
