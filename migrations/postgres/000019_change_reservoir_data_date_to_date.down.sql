-- Revert reservoir_data.date from DATE back to TIMESTAMPTZ
ALTER TABLE reservoir_data
    ALTER COLUMN date TYPE TIMESTAMPTZ USING date::TIMESTAMPTZ;
