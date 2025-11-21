-- Индекс удаляется первым
DROP INDEX IF EXISTS idx_files_category_created_at;

-- Затем удаляются таблицы в обратном порядке от их создания
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS categories;