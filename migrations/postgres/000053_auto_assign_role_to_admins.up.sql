CREATE OR REPLACE FUNCTION assign_new_role_to_admins()
    RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO users_roles (user_id, role_id)
    SELECT ur.user_id, NEW.id
    FROM users_roles ur
             JOIN roles r ON r.id = ur.role_id
    WHERE r.name = 'admin'
    ON CONFLICT DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_assign_new_role_to_admins
    AFTER INSERT ON roles
    FOR EACH ROW
EXECUTE FUNCTION assign_new_role_to_admins();
