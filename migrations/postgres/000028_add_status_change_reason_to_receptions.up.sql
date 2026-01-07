ALTER TABLE receptions 
    ADD COLUMN status_change_reason TEXT,
    ADD COLUMN informed BOOLEAN DEFAULT FALSE NOT NULL,
    ADD COLUMN informed_by_user_id BIGINT REFERENCES users(id);