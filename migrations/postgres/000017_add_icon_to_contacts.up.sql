ALTER TABLE Contacts
    ADD COLUMN icon_id BIGINT REFERENCES files(id) ON DELETE SET NULL;

COMMENT ON COLUMN Contacts.icon_id IS 'Foreign key to files table for contact icon';
