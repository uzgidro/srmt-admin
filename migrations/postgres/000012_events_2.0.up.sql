-- Migration 000012: Events 2.0 - Add event types, statuses, and file links

-- Create event_status table
CREATE TABLE IF NOT EXISTS event_status (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT
);

-- Insert predefined event statuses
INSERT INTO event_status (name, description) VALUES
    ('Draft', 'Event is being prepared and not yet confirmed'),
    ('Planned', 'Event is scheduled and confirmed'),
    ('Active', 'Event is currently ongoing or ready to start'),
    ('Completed', 'Event has finished successfully'),
    ('Cancelled', 'Event was planned but cancelled'),
    ('Postponed', 'Event is delayed to a future date')
ON CONFLICT (name) DO NOTHING;

-- Create event_type table
CREATE TABLE IF NOT EXISTS event_type (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT
);

-- Insert predefined event types
INSERT INTO event_type (name, description) VALUES
    ('Meeting', 'Business meetings, conferences, and discussions'),
    ('Training', 'Training sessions, workshops, and educational events'),
    ('Inspection', 'Inspections, audits, and compliance checks'),
    ('Maintenance', 'Maintenance activities and technical work')
ON CONFLICT (name) DO NOTHING;

-- Add new columns to events table
ALTER TABLE events ADD COLUMN IF NOT EXISTS event_status_id INTEGER;
ALTER TABLE events ADD COLUMN IF NOT EXISTS event_type_id INTEGER;

-- Set default values for existing rows (Active status = 3, Meeting type = 1)
UPDATE events SET event_status_id = 3 WHERE event_status_id IS NULL;
UPDATE events SET event_type_id = 1 WHERE event_type_id IS NULL;

-- Make created_by_user_id NOT NULL after setting defaults
UPDATE events SET created_by_user_id = 1 WHERE created_by_user_id IS NULL;

-- Add NOT NULL constraints and foreign keys
ALTER TABLE events ALTER COLUMN name SET NOT NULL;
ALTER TABLE events ALTER COLUMN event_date SET NOT NULL;
ALTER TABLE events ALTER COLUMN responsible_contact_id SET NOT NULL;
ALTER TABLE events ALTER COLUMN created_by_user_id SET NOT NULL;
ALTER TABLE events ALTER COLUMN event_status_id SET NOT NULL;
ALTER TABLE events ALTER COLUMN event_type_id SET NOT NULL;

-- Add foreign key constraints
ALTER TABLE events ADD CONSTRAINT fk_events_status
    FOREIGN KEY (event_status_id) REFERENCES event_status(id) ON DELETE RESTRICT;

ALTER TABLE events ADD CONSTRAINT fk_events_type
    FOREIGN KEY (event_type_id) REFERENCES event_type(id) ON DELETE RESTRICT;

-- Create event_file_links junction table for many-to-many relationship
CREATE TABLE IF NOT EXISTS event_file_links (
    event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, file_id)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_events_status ON events(event_status_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type_id);
CREATE INDEX IF NOT EXISTS idx_events_date ON events(event_date);
CREATE INDEX IF NOT EXISTS idx_events_organization ON events(organization_id);
CREATE INDEX IF NOT EXISTS idx_event_file_links_event ON event_file_links(event_id);
CREATE INDEX IF NOT EXISTS idx_event_file_links_file ON event_file_links(file_id);

-- Drop old attachment_path column (deprecated in favor of file links)
ALTER TABLE events DROP COLUMN IF EXISTS attachment_path;
