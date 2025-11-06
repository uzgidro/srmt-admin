CREATE TABLE reservoir_device_summary (
                                        id BIGSERIAL PRIMARY KEY,

                                        organization_id BIGINT NOT NULL REFERENCES Organizations(id) ON DELETE RESTRICT,

                                        device_type_name VARCHAR(255) NOT NULL DEFAULT 'default',

                                        count_total INTEGER NOT NULL DEFAULT 0,

                                        count_installed INTEGER NOT NULL DEFAULT 0,

                                        count_operational INTEGER NOT NULL DEFAULT 0,

                                        count_faulty INTEGER NOT NULL DEFAULT 0,

                                        count_active INTEGER NOT NULL DEFAULT 0,

                                        count_automation_scope INTEGER NOT NULL DEFAULT 0,

                                        criterion_1 NUMERIC,
                                        criterion_2 NUMERIC,

                                        created_at TIMESTAMPTZ DEFAULT NOW(),
                                        updated_at TIMESTAMPTZ,
                                        updated_by_user_id BIGINT REFERENCES Users(id) ON DELETE SET NULL,

                                        CONSTRAINT uq_org_device_type
                                            UNIQUE (organization_id, device_type_name)
);


CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON reservoir_device_summary
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- --- 3. Индексы ---
CREATE INDEX idx_reservoir_device_summary_org_id ON reservoir_device_summary(organization_id);