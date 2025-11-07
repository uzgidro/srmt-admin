CREATE TABLE IF NOT EXISTS Departments (
                             id BIGSERIAL PRIMARY KEY,
                             name VARCHAR(255) NOT NULL,
                             description TEXT,

                             organization_id BIGINT NOT NULL REFERENCES Organizations(id) ON DELETE RESTRICT,
                             created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                             updated_at TIMESTAMPTZ
);

CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON Departments
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

CREATE INDEX IF NOT EXISTS idx_departments_organization_id ON Departments(organization_id);

ALTER TABLE Contacts

    ADD COLUMN dob DATE,

    ADD COLUMN department_id BIGINT REFERENCES Departments(id) ON DELETE SET NULL;

CREATE INDEX idx_contacts_department_id ON Contacts(department_id);