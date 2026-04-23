ALTER TABLE ges_config
    ADD COLUMN max_daily_production_mln_kwh NUMERIC NOT NULL DEFAULT 0
        CHECK (max_daily_production_mln_kwh >= 0);
