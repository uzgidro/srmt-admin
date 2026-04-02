DROP INDEX IF EXISTS idx_one_ongoing_discharge_per_org;

-- Восстанавливаем старый индекс
CREATE INDEX idx_discharges_ongoing ON idle_water_discharges (id) WHERE end_time IS NULL;
