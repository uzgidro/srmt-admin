CREATE INDEX idx_shutdowns_ongoing ON shutdowns (id) WHERE end_time IS NULL;
CREATE INDEX idx_discharges_ongoing ON idle_water_discharges (id) WHERE end_time IS NULL;
