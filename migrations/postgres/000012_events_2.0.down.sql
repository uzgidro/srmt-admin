-- Rollback migration 000012: Events 2.0

-- Drop indexes
DROP INDEX IF EXISTS idx_event_file_links_file;
DROP INDEX IF EXISTS idx_event_file_links_event;
DROP INDEX IF EXISTS idx_events_organization;
DROP INDEX IF EXISTS idx_events_date;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_status;

-- Drop event_file_links junction table
DROP TABLE IF EXISTS event_file_links;

-- Re-add attachment_path column
ALTER TABLE events ADD COLUMN IF NOT EXISTS attachment_path TEXT;

-- Drop foreign key constraints
ALTER TABLE events DROP CONSTRAINT IF EXISTS fk_events_type;
ALTER TABLE events DROP CONSTRAINT IF EXISTS fk_events_status;

-- Remove NOT NULL constraints
ALTER TABLE events ALTER COLUMN event_type_id DROP NOT NULL;
ALTER TABLE events ALTER COLUMN event_status_id DROP NOT NULL;
ALTER TABLE events ALTER COLUMN created_by_user_id DROP NOT NULL;
ALTER TABLE events ALTER COLUMN responsible_contact_id DROP NOT NULL;
ALTER TABLE events ALTER COLUMN event_date DROP NOT NULL;
ALTER TABLE events ALTER COLUMN name DROP NOT NULL;

-- Drop new columns
ALTER TABLE events DROP COLUMN IF EXISTS event_type_id;
ALTER TABLE events DROP COLUMN IF EXISTS event_status_id;

-- Drop event_type table
DROP TABLE IF EXISTS event_type;

-- Drop event_status table
DROP TABLE IF EXISTS event_status;
