CREATE TABLE idle_water_discharges
(
    id              SERIAL PRIMARY KEY,
    organization_id INTEGER     NOT NULL REFERENCES organizations (id) ON DELETE RESTRICT,
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ,
    flow_rate_m3_s       NUMERIC     NOT NULL,
    reason          TEXT,
    created_by   INTEGER     NOT NULL REFERENCES users (id),
    approved_by  INTEGER REFERENCES users (id),
    approved        BOOLEAN,
    CONSTRAINT check_end_time
        CHECK (end_time IS NULL OR end_time > start_time)
);

CREATE INDEX idx_idle_discharges_org_id ON idle_water_discharges (organization_id);

DROP VIEW IF EXISTS v_idle_water_discharges_with_volume;

CREATE VIEW v_idle_water_discharges_with_volume AS
SELECT id,
       organization_id,
       start_time,
       end_time,
       flow_rate_m3_s,
       reason,
       created_by,
       approved_by,
       approved,

       (end_time IS NULL) AS is_ongoing,

       (
           EXTRACT(EPOCH FROM
                   (COALESCE(end_time, NOW()) - start_time)
           ) * flow_rate_m3_s / 1000000
           )              AS total_volume_mln_m3

FROM idle_water_discharges;