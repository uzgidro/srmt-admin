CREATE TABLE IF NOT EXISTS users (
                                     id BIGSERIAL PRIMARY KEY,
                                     name VARCHAR(255) NOT NULL UNIQUE,
                                     pass_hash TEXT NOT NULL,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roles (
                                     id BIGSERIAL PRIMARY KEY,
                                     name VARCHAR(255) NOT NULL UNIQUE,
                                     description TEXT
);

CREATE TABLE IF NOT EXISTS users_roles (
                                           user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                           role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
                                           PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS reservoirs (
                                          id BIGSERIAL PRIMARY KEY,
                                          name TEXT NOT NULL UNIQUE,
                                          position INTEGER
);

CREATE TABLE IF NOT EXISTS level_volume (
                                            id BIGSERIAL PRIMARY KEY,
                                            level DOUBLE PRECISION NOT NULL,
                                            volume DOUBLE PRECISION NOT NULL,
                                            res_id BIGINT NOT NULL REFERENCES reservoirs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS data (
                                    id BIGSERIAL PRIMARY KEY,
                                    level DOUBLE PRECISION,
                                    volume DOUBLE PRECISION,
                                    "release" DOUBLE PRECISION,
                                    income DOUBLE PRECISION,
                                    res_id BIGINT NOT NULL REFERENCES reservoirs(id) ON DELETE CASCADE,
                                    date DATE NOT NULL,
                                    UNIQUE (res_id, date)
);

CREATE TABLE IF NOT EXISTS indicator_height (
                                                id BIGSERIAL PRIMARY KEY,
                                                height DOUBLE PRECISION NOT NULL,
                                                res_id BIGINT NOT NULL REFERENCES reservoirs(id) ON DELETE CASCADE
);