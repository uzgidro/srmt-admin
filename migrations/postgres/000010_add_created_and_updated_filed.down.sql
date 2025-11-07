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
                EXECUTE format('DROP TRIGGER IF EXISTS set_timestamp ON %I;', t_name);

                EXECUTE format('
            ALTER TABLE %I
                DROP COLUMN IF EXISTS created_at,
                DROP COLUMN IF EXISTS updated_at;
        ', t_name);
            END LOOP;
    END;
$$;