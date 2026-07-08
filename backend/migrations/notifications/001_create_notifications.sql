-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS notifications;

-- Notifications: tracks each notification to be sent
CREATE TABLE notifications.notifications (
    id              UUID         PRIMARY KEY,
    recipient_type  VARCHAR(20)  NOT NULL,
    recipient_id    UUID         NOT NULL,
    notif_type      VARCHAR(50)  NOT NULL,
    channel         VARCHAR(20)  NOT NULL,
    title           VARCHAR(255) NOT NULL,
    title_ar        VARCHAR(255),
    body            TEXT         NOT NULL,
    body_ar         TEXT,
    data            JSONB,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    priority        VARCHAR(20)  NOT NULL DEFAULT 'normal',
    retry_count     INTEGER      NOT NULL DEFAULT 0,
    max_retries     INTEGER      NOT NULL DEFAULT 3,
    last_error      TEXT,
    scheduled_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_notif_recipient CHECK (recipient_type IN ('user', 'driver', 'merchant')),
    CONSTRAINT chk_notif_channel CHECK (channel IN ('push', 'sms', 'email')),
    CONSTRAINT chk_notif_status CHECK (status IN ('pending', 'sending', 'sent', 'delivered', 'failed', 'cancelled')),
    CONSTRAINT chk_notif_priority CHECK (priority IN ('normal', 'high')),
    CONSTRAINT chk_notif_retries_nonneg CHECK (retry_count >= 0)
);
CREATE INDEX idx_notif_recipient ON notifications.notifications (recipient_type, recipient_id, created_at DESC);
CREATE INDEX idx_notif_pending ON notifications.notifications (scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_notif_status ON notifications.notifications (status);

-- Preferences: per-recipient notification preferences
CREATE TABLE notifications.preferences (
    id              UUID         PRIMARY KEY,
    recipient_type  VARCHAR(20)  NOT NULL,
    recipient_id    UUID         NOT NULL,
    phone_number    VARCHAR(30),
    email           VARCHAR(255),
    device_tokens   TEXT[],
    prefs           JSONB        NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(recipient_type, recipient_id),
    CONSTRAINT chk_pref_recipient CHECK (recipient_type IN ('user', 'driver', 'merchant'))
);
CREATE INDEX idx_pref_recipient ON notifications.preferences (recipient_type, recipient_id);

-- Outbox
CREATE TABLE IF NOT EXISTS notifications.outbox (
    id                 BIGSERIAL    PRIMARY KEY,
    event_id           UUID         NOT NULL UNIQUE,
    event_type         TEXT         NOT NULL,
    event_version      INTEGER      NOT NULL DEFAULT 1,
    schema_version     INTEGER      NOT NULL DEFAULT 1,
    payload            JSONB        NOT NULL,
    occurred_at        TIMESTAMPTZ  NOT NULL,
    producer           TEXT         NOT NULL,
    correlation_id     TEXT,
    trace_id           TEXT,
    actor_type         TEXT,
    actor_id           TEXT,
    actor_ip           TEXT,
    actor_user_agent   TEXT,
    published_at       TIMESTAMPTZ,
    retry_count        INTEGER      NOT NULL DEFAULT 0,
    next_retry_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_error         TEXT,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_outbox_pending
    ON notifications.outbox (next_retry_at)
    WHERE published_at IS NULL;

-- Inbox (for idempotent event consumption)
CREATE TABLE IF NOT EXISTS notifications.inbox (
    event_id      UUID         NOT NULL,
    handler_name  VARCHAR(100) NOT NULL,
    event_type    TEXT,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, handler_name)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notifications.inbox;
DROP TABLE IF EXISTS notifications.outbox;
DROP TABLE IF EXISTS notifications.preferences;
DROP TABLE IF EXISTS notifications.notifications;
-- +goose StatementEnd
