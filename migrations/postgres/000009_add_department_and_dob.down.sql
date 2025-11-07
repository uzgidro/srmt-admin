-- --- 1. Модифицируем таблицу "Contacts" (удаляем колонки) ---
ALTER TABLE Contacts
    DROP COLUMN IF EXISTS dob,
    DROP COLUMN IF EXISTS department_id;
-- (Индекс idx_contacts_department_id удалится автоматически)

---

-- --- 2. Удаляем таблицу "Departments" ---
DROP TRIGGER IF EXISTS set_timestamp ON Departments;
DROP TABLE IF EXISTS Departments;