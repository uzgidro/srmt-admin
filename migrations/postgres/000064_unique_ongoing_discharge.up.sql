-- Закрываем дублирующиеся ongoing сбросы: для каждой организации оставляем
-- только самый новый (по id), остальным ставим end_time = start_time следующего сброса
-- (для миграции используем NOW() как разумное приближение)
UPDATE idle_water_discharges
SET end_time = NOW()
WHERE end_time IS NULL
  AND id NOT IN (
      SELECT MAX(id)
      FROM idle_water_discharges
      WHERE end_time IS NULL
      GROUP BY organization_id
  );

-- Уникальный partial index: максимум 1 ongoing сброс на организацию
CREATE UNIQUE INDEX idx_one_ongoing_discharge_per_org
    ON idle_water_discharges (organization_id) WHERE end_time IS NULL;

-- Старый индекс больше не нужен (новый покрывает)
DROP INDEX IF EXISTS idx_discharges_ongoing;
