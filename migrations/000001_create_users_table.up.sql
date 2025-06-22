-- Создание таблицы users
CREATE TABLE IF NOT EXISTS users (
                       id INTEGER PRIMARY KEY AUTOINCREMENT, -- SERIAL для PostgreSQL, INTEGER PRIMARY KEY AUTOINCREMENT для SQLite
                       name VARCHAR(255) NOT NULL UNIQUE,
                       pass_hash TEXT NOT NULL, -- TEXT для SQLite, VARCHAR/TEXT для PostgreSQL
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы roles
CREATE TABLE IF NOT EXISTS roles (
                       id INTEGER PRIMARY KEY AUTOINCREMENT, -- SERIAL для PostgreSQL, INTEGER PRIMARY KEY AUTOINCREMENT для SQLite
                       name VARCHAR(255) NOT NULL UNIQUE,
                       description TEXT -- TEXT для SQLite, VARCHAR/TEXT для PostgreSQL
);

-- Создание таблицы user_roles (таблица связей многие-ко-многим)
CREATE TABLE IF NOT EXISTS user_roles (
                            user_id INTEGER NOT NULL,
                            role_id INTEGER NOT NULL,
                            PRIMARY KEY (user_id, role_id), -- Составной первичный ключ
                            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
                            FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);