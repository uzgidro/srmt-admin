CREATE TABLE positions (
                           id SERIAL PRIMARY KEY,
                           name VARCHAR(255) NOT NULL UNIQUE,
                           description TEXT
);

CREATE TABLE organizations (
                               id SERIAL PRIMARY KEY,
                               name VARCHAR(255) NOT NULL,
                               parent_organization_id INTEGER REFERENCES Organizations(id) ON DELETE SET NULL
);

CREATE TABLE organization_types (
                                   id SERIAL PRIMARY KEY,
                                   name VARCHAR(100) NOT NULL UNIQUE,
                                   description TEXT
);

ALTER TABLE users
    ADD COLUMN fio VARCHAR(255) UNIQUE,
    ADD COLUMN email VARCHAR(255) UNIQUE,
    ADD COLUMN phone VARCHAR(50) UNIQUE,
    ADD COLUMN ip_phone VARCHAR(50) UNIQUE,
    ADD COLUMN position_id INTEGER REFERENCES Positions(id) ON DELETE SET NULL,
    ADD COLUMN organization_id INTEGER REFERENCES Organizations(id) ON DELETE CASCADE;

CREATE TABLE organization_type_links (
                                         organization_id INTEGER NOT NULL REFERENCES Organizations(id) ON DELETE CASCADE,
                                         type_id INTEGER NOT NULL REFERENCES organization_types(id) ON DELETE CASCADE,

                                         PRIMARY KEY (organization_id, type_id)
);