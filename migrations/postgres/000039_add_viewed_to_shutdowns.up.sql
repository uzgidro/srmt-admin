ALTER TABLE shutdowns ADD COLUMN viewed BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_shutdowns_viewed ON shutdowns(viewed) WHERE viewed = FALSE;
