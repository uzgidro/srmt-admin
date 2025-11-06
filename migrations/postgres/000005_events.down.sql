-- 1. Удаляем триггеры
DROP TRIGGER IF EXISTS set_timestamp ON users;
DROP TRIGGER IF EXISTS set_timestamp ON Contacts;

-- 2. Удаляем ограничения и колонки из "users"
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS fk_users_contact,
    DROP CONSTRAINT IF EXISTS uq_users_contact_id,
    DROP COLUMN IF EXISTS contact_id,
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS updated_at;

-- 3. Переименовываем 'login' обратно в 'name'
-- (Данные ФИО сохранятся, т.к. они и были в этой колонке)
ALTER TABLE users
    RENAME COLUMN login TO name;

-- 4. Удаляем таблицу Contacts
DROP TABLE IF EXISTS Contacts;

-- (Функцию trigger_set_timestamp() можно не удалять,
-- она может использоваться другими таблицами)

-- 1. Удаляем триггер
DROP TRIGGER IF EXISTS set_timestamp ON Events;

-- 2. Удаляем таблицу
DROP TABLE IF EXISTS Events;