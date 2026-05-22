-- Revert 000081.
--
-- Note: dropping user_organizations loses any multi-org bindings added
-- after the migration. The role rename is reverted only if reservoir_duty
-- does not already exist.

UPDATE roles SET name = 'reservoir_duty'
WHERE name = 'reservoir_flood'
  AND NOT EXISTS (SELECT 1 FROM roles WHERE name = 'reservoir_duty');

DROP TABLE IF EXISTS user_organizations;
