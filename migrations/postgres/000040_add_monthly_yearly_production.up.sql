ALTER TABLE ges_production
    ADD COLUMN monthly_energy_production DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN yearly_energy_production DOUBLE PRECISION NOT NULL DEFAULT 0;
