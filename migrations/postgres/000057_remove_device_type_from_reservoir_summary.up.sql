-- Remove device_type_name from reservoir_device_summary
-- One row per organization instead of per org+device_type

ALTER TABLE reservoir_device_summary DROP CONSTRAINT IF EXISTS uq_org_device_type;
ALTER TABLE reservoir_device_summary DROP COLUMN IF EXISTS device_type_name;
ALTER TABLE reservoir_device_summary ADD CONSTRAINT uq_reservoir_device_org UNIQUE (organization_id);
