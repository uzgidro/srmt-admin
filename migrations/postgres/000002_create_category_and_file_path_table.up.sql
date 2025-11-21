CREATE TABLE IF NOT EXISTS categories (
                                          id BIGSERIAL PRIMARY KEY,
                                          parent_id BIGINT REFERENCES categories(id) ON DELETE SET NULL, -- При удалении родителя, дочерние станут корневыми
                                          name VARCHAR(255) NOT NULL UNIQUE,
                                          display_name TEXT NOT NULL,
                                          description TEXT
);

CREATE TABLE IF NOT EXISTS files (
                                     id BIGSERIAL PRIMARY KEY,
                                     file_name TEXT NOT NULL,
                                     object_key TEXT NOT NULL UNIQUE,
                                     category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE, -- При удалении категории, удаляются и файлы
                                     mime_type VARCHAR(255),
                                     size_bytes BIGINT,
                                     created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индекс для ускорения поиска последних файлов в категориях
CREATE INDEX IF NOT EXISTS idx_files_category_created_at ON files (category_id, created_at DESC);