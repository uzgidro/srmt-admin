DROP TABLE IF EXISTS solar_production_plan;
DROP TABLE IF EXISTS solar_daily_data;
DROP TABLE IF EXISTS solar_config;

ALTER TABLE ges_daily_data DROP COLUMN IF EXISTS own_consumption_kwh;
