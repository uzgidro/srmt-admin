CREATE TABLE cascade_daily_data (
    id                BIGSERIAL PRIMARY KEY,
    organization_id   BIGINT NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    date              DATE NOT NULL,
    temperature       NUMERIC,
    weather_condition TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, date)
);

CREATE INDEX idx_cascade_daily_data_date ON cascade_daily_data(date);

-- Backfill: move weather only from cascade-level rows (org present in cascade_config).
-- Station-level "noise" rows are ignored.
INSERT INTO cascade_daily_data (organization_id, date, temperature, weather_condition)
SELECT gdd.organization_id, gdd.date, gdd.temperature, gdd.weather_condition
FROM ges_daily_data gdd
WHERE gdd.organization_id IN (SELECT organization_id FROM cascade_config)
  AND (gdd.temperature IS NOT NULL OR gdd.weather_condition IS NOT NULL)
ON CONFLICT (organization_id, date) DO NOTHING;

ALTER TABLE ges_daily_data
    DROP COLUMN temperature,
    DROP COLUMN weather_condition;
