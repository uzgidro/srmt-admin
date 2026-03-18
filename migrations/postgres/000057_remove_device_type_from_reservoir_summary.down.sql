ALTER TABLE reservoir_device_summary DROP CONSTRAINT IF EXISTS uq_reservoir_device_org;
ALTER TABLE reservoir_device_summary ADD COLUMN device_type_name VARCHAR(255) NOT NULL DEFAULT 'default';
ALTER TABLE reservoir_device_summary ADD CONSTRAINT uq_org_device_type UNIQUE (organization_id, device_type_name);
