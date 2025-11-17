-- Remove NOT NULL constraint from organization_id in Incidents table
-- This allows incidents to be associated with all organizations or no specific organization

ALTER TABLE Incidents
    ALTER COLUMN organization_id DROP NOT NULL;
