-- Справочник категорий инфраструктурных событий
CREATE TABLE sc_infra_event_categories (
    id           BIGSERIAL PRIMARY KEY,
    slug         VARCHAR(50) NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    label        TEXT NOT NULL,
    sort_order   INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO sc_infra_event_categories (slug, display_name, label, sort_order) VALUES
    ('video', 'Видеонаблюдение', 'Vaziyatlar markaziga integratsiya qilingan Tizim tashkilotlari Videokuzatuv tizimi holati', 1),
    ('comms', 'Связь', 'Tizim tashkilotlari Aloqa tizimi holati', 2),
    ('ascue', 'АСКУЭ', 'ASKUE holati to''g''risida ma''lumot', 3),
    ('atnt', 'АТНТ', 'ATNT holati to''g''risida ma''lumot', 4),
    ('observation', 'Наблюдения', 'Tizim tashkilotlarida kuzatilgan holatlar', 5);

-- События инфраструктуры
CREATE TABLE sc_infra_events (
    id                  BIGSERIAL PRIMARY KEY,
    category_id         BIGINT NOT NULL REFERENCES sc_infra_event_categories(id) ON DELETE RESTRICT,
    organization_id     BIGINT NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    occurred_at         TIMESTAMPTZ NOT NULL,
    restored_at         TIMESTAMPTZ,
    description         TEXT NOT NULL,
    remediation         TEXT,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ,

    CONSTRAINT chk_restored_after_occurred CHECK (restored_at IS NULL OR restored_at > occurred_at)
);

CREATE INDEX idx_sc_infra_events_category_occurred ON sc_infra_events(category_id, occurred_at);
CREATE INDEX idx_sc_infra_events_occurred ON sc_infra_events(occurred_at);
CREATE INDEX idx_sc_infra_events_org ON sc_infra_events(organization_id);
CREATE INDEX idx_sc_infra_events_unresolved ON sc_infra_events(id) WHERE restored_at IS NULL;

CREATE TRIGGER set_timestamp_sc_infra_events BEFORE UPDATE ON sc_infra_events
    FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- Привязка файлов к событиям
CREATE TABLE sc_infra_event_file_links (
    event_id    BIGINT NOT NULL REFERENCES sc_infra_events(id) ON DELETE CASCADE,
    file_id     BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, file_id)
);

CREATE INDEX idx_sc_infra_event_file_links_file ON sc_infra_event_file_links(file_id);
