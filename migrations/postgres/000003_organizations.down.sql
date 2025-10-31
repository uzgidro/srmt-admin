DROP TABLE IF EXISTS organization_type_links;

ALTER TABLE users
    DROP COLUMN IF EXISTS organization_id,
    DROP COLUMN IF EXISTS position_id,
    DROP COLUMN IF EXISTS ip_phone,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS fio;

DROP TABLE IF EXISTS organization_types;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS positions;