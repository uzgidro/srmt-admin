-- Remove unique constraint
ALTER TABLE reservoir_data
    DROP CONSTRAINT reservoir_data_org_date_unique;
