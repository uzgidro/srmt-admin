-- Add unique constraint on organization_id and date to support upsert operations
ALTER TABLE reservoir_data
    ADD CONSTRAINT reservoir_data_org_date_unique UNIQUE (organization_id, date);
