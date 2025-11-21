-- --- Вспомогательная функция (если ее еще нет) ---
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


-- --- 1. Создаем Contacts ---
-- (Используем BIGSERIAL, как у вас в users, для единообразия)
CREATE TABLE contacts (
                          id BIGSERIAL PRIMARY KEY,
                          fio VARCHAR(255) NOT NULL,
                          phone VARCHAR(50) UNIQUE,
                          ip_phone VARCHAR(50) UNIQUE,
                          email VARCHAR(255) UNIQUE,

                          position_id INTEGER REFERENCES Positions(id) ON DELETE SET NULL,
                          organization_id INTEGER REFERENCES Organizations(id) ON DELETE SET NULL,
                          external_organization_name VARCHAR(255),

                          created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                          updated_at TIMESTAMPTZ
);

-- Триггер для Contacts
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON Contacts
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();


-- --- 2. Модернизируем существующую таблицу "users" ---

-- Добавляем новые колонки (пока NULLABLE)
ALTER TABLE users
    ADD COLUMN contact_id BIGINT, -- (BIGINT, т.к. ссылается на BIGSERIAL)
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN updated_at TIMESTAMPTZ;

-- --- 3. Миграция данных ---

-- 3.1. Создаем по одному "Контакту" для каждого "Пользователя"
-- (Берем ФИО из старой колонки 'name')
INSERT INTO Contacts (fio)
SELECT name FROM users;

-- 3.2. Связываем существующих "Пользователей" с их новыми "Контактами"
UPDATE users
SET contact_id = Contacts.id
FROM Contacts
WHERE users.name = Contacts.fio;

-- --- 4. Завершаем модернизацию "users" ---

-- 4.1. Переименовываем 'name' (которая теперь логин)
ALTER TABLE users
    RENAME COLUMN name TO login;

-- 4.2. Добавляем ограничения, которые не могли добавить раньше
ALTER TABLE users
    -- Делаем 'contact_id' обязательным
    ALTER COLUMN contact_id SET NOT NULL,

    -- Добавляем внешний ключ
    ADD CONSTRAINT fk_users_contact
        FOREIGN KEY (contact_id) REFERENCES Contacts(id) ON DELETE CASCADE,

    -- Добавляем UNIQUE (у одного контакта - один логин)
    ADD CONSTRAINT uq_users_contact_id UNIQUE (contact_id);

-- 4.3. Добавляем триггер для 'users'
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON users
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();


-- --- 1. Таблица Events (События) ---

CREATE TABLE Events (
                        id BIGSERIAL PRIMARY KEY,

    -- Название (название, тема)
                        name VARCHAR(255) NOT NULL,

    -- Место (место проведения)
                        location VARCHAR(255),

    -- Дата (дата и время самого события)
                        event_date TIMESTAMPTZ NOT NULL,

    -- Описание (описание)
                        description TEXT,

    -- Приложение (путь к файлу)
                        attachment_path TEXT,

    -- "Организатор" (какая организация проводит)
                        organization_id BIGINT REFERENCES Organizations(id) ON DELETE SET NULL,

    -- "Ответственный" (ссылка на КОНТАКТ)
                        responsible_contact_id BIGINT REFERENCES Contacts(id) ON DELETE SET NULL,

    -- --- Стандартные поля аудита ---
                        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                        created_by_user_id BIGINT REFERENCES Users(id) ON DELETE SET NULL,
                        updated_at TIMESTAMPTZ,
                        updated_by_user_id BIGINT REFERENCES Users(id) ON DELETE SET NULL
);

-- --- 2. Триггер для updated_at ---
-- (Используем ту же функцию trigger_set_timestamp(),
-- которую создавали для других таблиц)

CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON Events
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- --- 3. Индексы ---

CREATE INDEX idx_events_event_date ON Events(event_date);
CREATE INDEX idx_events_organization_id ON Events(organization_id);
CREATE INDEX idx_events_responsible_contact_id ON Events(responsible_contact_id);