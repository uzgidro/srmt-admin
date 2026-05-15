-- Useful water consumption (m³/s) for GES daily data: irrigation, drinking
-- water, etc. Subtracted from idle discharge (total_outflow - ges_flow) when
-- the daily report is generated. Nullable; non-negative when present.
ALTER TABLE ges_daily_data
    ADD COLUMN consumption_m3_s NUMERIC
        CHECK (consumption_m3_s IS NULL OR consumption_m3_s >= 0);
