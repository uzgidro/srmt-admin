-- 000081: multi-organization access for users.
--
-- Until now a user belonged to exactly one organization (via
-- contacts.organization_id). Duty operators had to keep two accounts — one
-- for their HPP cascade, one for their reservoir. This adds a many-to-many
-- user_organizations table so a single profile can be bound to several
-- organizations. contacts.organization_id is left intact (other modules
-- still read it); user_organizations becomes the source of truth for
-- access checks.

CREATE TABLE IF NOT EXISTS user_organizations (
    user_id         BIGINT  NOT NULL REFERENCES users(id)         ON DELETE CASCADE,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, organization_id)
);

CREATE INDEX IF NOT EXISTS idx_user_organizations_org
    ON user_organizations (organization_id);

-- Backfill: carry over each user's current single organization.
INSERT INTO user_organizations (user_id, organization_id)
SELECT u.id, c.organization_id
FROM users u
JOIN contacts c ON u.contact_id = c.id
WHERE c.organization_id IS NOT NULL
ON CONFLICT DO NOTHING;

-- Rename role reservoir_duty -> reservoir_flood (idempotent). role_id is
-- preserved by UPDATE, so users_roles links stay intact. If the role was
-- never seeded (it lived only in code), INSERT it under the new name.
-- The NOT EXISTS guard on the UPDATE avoids a UNIQUE(name) violation in the
-- edge case where both reservoir_duty and reservoir_flood already exist.
UPDATE roles SET name = 'reservoir_flood'
WHERE name = 'reservoir_duty'
  AND NOT EXISTS (SELECT 1 FROM roles WHERE name = 'reservoir_flood');
INSERT INTO roles (name)
SELECT 'reservoir_flood'
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'reservoir_flood');
