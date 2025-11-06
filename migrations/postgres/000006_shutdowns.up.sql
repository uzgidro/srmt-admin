-- --- 1. Создаем основную таблицу "Shutdowns" (Аварийные отключения) ---

CREATE TABLE Shutdowns (
                           id BIGSERIAL PRIMARY KEY,

    -- "Организация" (Какая ГЭС)
                           organization_id BIGINT NOT NULL REFERENCES Organizations(id) ON DELETE RESTRICT,

    -- "Начало" и "Конец"
                           start_time TIMESTAMPTZ NOT NULL,
                           end_time TIMESTAMPTZ,

    -- "Невыработанная мощность" (в МВт*ч)
                           generation_loss_mwh NUMERIC,

    -- "Причина" (текстом)
                           reason TEXT,

    -- "Ссылка на холостой сброс" (если он был)
                           idle_discharge_id BIGINT UNIQUE REFERENCES idle_water_discharges(id) ON DELETE SET NULL,

    -- "Кто доложил" (нач. смены - ссылка на Contacts)
                           reported_by_contact_id BIGINT REFERENCES contacts(id) ON DELETE SET NULL,

    -- --- Стандартные поля аудита (Кто и когда внес) ---
                           created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                           created_by_user_id BIGINT REFERENCES Users(id) ON DELETE SET NULL,
                           updated_at TIMESTAMPTZ,

    -- --- Ограничения ---
                           CONSTRAINT check_shutdown_times
                               CHECK (end_time IS NULL OR end_time > start_time)
);

-- --- 2. Создаем триггер для авто-обновления `updated_at` ---

CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON Shutdowns
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- --- 3. Индексы для быстрого поиска ---

CREATE INDEX idx_shutdowns_organization_id ON Shutdowns(organization_id);
CREATE INDEX idx_shutdowns_start_time ON Shutdowns(start_time);
CREATE INDEX idx_shutdowns_reported_by_contact_id ON Shutdowns(reported_by_contact_id);
CREATE INDEX idx_shutdowns_idle_discharge_id ON Shutdowns(idle_discharge_id);


-- --- 4. (РЕКОМЕНДУЕТСЯ) VIEW с расчетом длительности ---

CREATE VIEW V_Shutdowns AS
SELECT
    s.*, -- (Выбираем все поля из Shutdowns)

    -- (Флаг) Идет ли отключение прямо сейчас
    (s.end_time IS NULL) AS is_ongoing,

    -- (Вычисление) Длительность в секундах
    EXTRACT(EPOCH FROM
            (COALESCE(s.end_time, NOW()) - s.start_time)
    ) AS duration_seconds
FROM
    Shutdowns s;


-- 1. Создаем таблицу "Incidents" (ЧП)

CREATE TABLE Incidents (
                           id BIGSERIAL PRIMARY KEY,

    -- "Организация"
                           organization_id BIGINT NOT NULL REFERENCES Organizations(id) ON DELETE RESTRICT,

    -- "Время" (когда произошло ЧП)
                           incident_time TIMESTAMPTZ NOT NULL,

    -- "Описание"
                           description TEXT,

    -- --- Поля аудита (Кто и когда внес) ---
                           created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                           created_by_user_id BIGINT REFERENCES Users(id) ON DELETE SET NULL,
                           updated_at TIMESTAMPTZ
);

-- 2. Триггер для "updated_at"
CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON Incidents
    FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- 3. Индексы
CREATE INDEX idx_incidents_organization_id ON Incidents(organization_id);
CREATE INDEX idx_incidents_incident_time ON Incidents(incident_time);