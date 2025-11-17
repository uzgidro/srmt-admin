-- Restore NOT NULL constraint on organization_id in Incidents table
-- Note: This will fail if there are any NULL values in organization_id

ALTER TABLE Incidents
    ALTER COLUMN organization_id SET NOT NULL;
