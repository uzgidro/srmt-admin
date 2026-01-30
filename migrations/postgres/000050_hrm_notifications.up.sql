-- HRM Notifications

CREATE TABLE IF NOT EXISTS hrm_notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Notification content
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    category VARCHAR(50) NOT NULL, -- vacation, document, training, review, task, system

    -- Link to related entity
    entity_type VARCHAR(50), -- vacation, document, training, review, etc.
    entity_id BIGINT,

    -- Priority
    priority VARCHAR(20) DEFAULT 'normal', -- low, normal, high, urgent

    -- Status
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,

    -- Action
    action_url VARCHAR(500), -- Deep link to relevant page
    action_label VARCHAR(100), -- Button text

    -- Delivery
    send_email BOOLEAN DEFAULT FALSE,
    email_sent_at TIMESTAMPTZ,

    -- Expiry
    expires_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_hrm_notifications_user ON hrm_notifications(user_id);
CREATE INDEX idx_hrm_notifications_read ON hrm_notifications(user_id, is_read) WHERE is_read = FALSE;
CREATE INDEX idx_hrm_notifications_category ON hrm_notifications(category);
CREATE INDEX idx_hrm_notifications_created ON hrm_notifications(created_at);
CREATE INDEX idx_hrm_notifications_entity ON hrm_notifications(entity_type, entity_id);

COMMENT ON TABLE hrm_notifications IS 'User notifications for HRM events';
