-- Fail early if weather-only rows exist (both user columns NULL)
DO $$ BEGIN
  IF EXISTS (
    SELECT 1 FROM ges_daily_data
    WHERE created_by_user_id IS NULL AND updated_by_user_id IS NULL
  ) THEN
    RAISE EXCEPTION 'Cannot roll back: rows exist with no user attribution (weather-only rows). Delete them first.';
  END IF;
END $$;

UPDATE ges_daily_data SET created_by_user_id = updated_by_user_id WHERE created_by_user_id IS NULL;
UPDATE ges_daily_data SET updated_by_user_id = created_by_user_id WHERE updated_by_user_id IS NULL;

ALTER TABLE ges_daily_data ALTER COLUMN created_by_user_id SET NOT NULL;
ALTER TABLE ges_daily_data ALTER COLUMN updated_by_user_id SET NOT NULL;
