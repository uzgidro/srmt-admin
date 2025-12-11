-- Change reservoir_data.date from TIMESTAMPTZ to DATE
ALTER TABLE reservoir_data
    ALTER COLUMN date TYPE DATE USING date::DATE;
