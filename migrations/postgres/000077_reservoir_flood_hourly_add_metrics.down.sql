ALTER TABLE reservoir_flood_hourly
    DROP COLUMN IF EXISTS capacity_mwt,
    DROP COLUMN IF EXISTS weather_condition,
    DROP COLUMN IF EXISTS temperature_c;
