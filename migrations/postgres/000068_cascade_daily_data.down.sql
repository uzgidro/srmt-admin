ALTER TABLE ges_daily_data
    ADD COLUMN temperature       NUMERIC,
    ADD COLUMN weather_condition TEXT;

UPDATE ges_daily_data gdd
SET temperature = cdd.temperature,
    weather_condition = cdd.weather_condition
FROM cascade_daily_data cdd
WHERE cdd.organization_id = gdd.organization_id
  AND cdd.date = gdd.date;

DROP TABLE IF EXISTS cascade_daily_data;
