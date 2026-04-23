-- Index on organizations.parent_organization_id.
--
-- Postgres does NOT auto-index foreign keys. Without this index,
-- the cascade-restricted GET /shutdowns query
--   WHERE (o.id = $3 OR o.parent_organization_id = $3)
-- degenerates into a sequential scan on `organizations` for the second
-- predicate, and the existing FK's ON DELETE SET NULL also pays a scan.
--
-- Adding this index:
--   - speeds up the cascade-shutdowns visibility filter;
--   - speeds up cascadefilter.Apply (handler-side, not SQL, but org tree
--     reads benefit too);
--   - speeds up FK cascade actions when an organization is deleted.

CREATE INDEX IF NOT EXISTS idx_organizations_parent_organization_id
    ON organizations(parent_organization_id);
