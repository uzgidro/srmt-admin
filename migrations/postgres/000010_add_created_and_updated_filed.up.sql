CREATE OR REPLACE FUNCTION trigger_set_timestamp()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


DO $$
    DECLARE
        t_name TEXT;
        table_list TEXT[] := ARRAY[
            'positions', 'categories', 'data', 'indicator_height',
            'level_volume', 'organization_types', 'organizations',
            'organization_type_links', 'reservoirs', 'roles', 'users_roles'
            ];
    BEGIN
        FOREACH t_name IN ARRAY table_list
            LOOP
                EXECUTE format('
            ALTER TABLE %I
                ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ;
        ', t_name);

                EXECUTE format('DROP TRIGGER IF EXISTS set_timestamp ON %I;', t_name);

                EXECUTE format('
            CREATE TRIGGER set_timestamp
                BEFORE UPDATE ON %I
                FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();
        ', t_name);
            END LOOP;
    END;
$$;