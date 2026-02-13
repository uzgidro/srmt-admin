DROP TRIGGER IF EXISTS set_timestamp_development_plans ON development_plans;
DROP TRIGGER IF EXISTS set_timestamp_trainings ON trainings;

DROP TABLE IF EXISTS development_goals;
DROP TABLE IF EXISTS development_plans;

ALTER TABLE training_participants DROP CONSTRAINT IF EXISTS fk_training_participants_certificate;

DROP TABLE IF EXISTS certificates;
DROP TABLE IF EXISTS training_participants;
DROP TABLE IF EXISTS trainings;
