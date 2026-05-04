ALTER TABLE reservoir_flood_hourly
    ADD COLUMN capacity_mwt       NUMERIC CHECK (capacity_mwt >= 0),
    ADD COLUMN weather_condition  TEXT,
    ADD COLUMN temperature_c      NUMERIC;
