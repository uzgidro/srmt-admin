DROP TRIGGER IF EXISTS set_timestamp ON Incidents;
DROP TABLE IF EXISTS Incidents;

-- 1. Удаляем VIEW
DROP VIEW IF EXISTS V_Shutdowns;

-- 2. Удаляем триггер
DROP TRIGGER IF EXISTS set_timestamp ON Shutdowns;

-- 3. Удаляем основную таблицу
DROP TABLE IF EXISTS Shutdowns;