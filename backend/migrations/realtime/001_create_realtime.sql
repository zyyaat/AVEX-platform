-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS realtime;

-- Inbox table for consumer-side idempotency (dedup of bus events).
-- Same schema as orders.inbox / dispatch.inbox.
CREATE TABLE IF NOT EXISTS realtime.inbox (
    event_id      UUID         NOT NULL,
    handler_name  VARCHAR(100) NOT NULL,
    event_type    TEXT,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, handler_name)
);

-- Optional: message buffer for offline clients.
-- When a client is offline, important messages (order status, wallet) can be
-- persisted here for catch-up on reconnect. Driver location updates are NOT
-- buffered (too high volume).
CREATE TABLE IF NOT EXISTS realtime.messages (
    id           UUID         PRIMARY KEY,
    channel      VARCHAR(100) NOT NULL,
    msg_type     VARCHAR(50)  NOT NULL,
    data         JSONB        NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ  NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);
CREATE INDEX idx_realtime_messages_channel ON realtime.messages (channel, created_at DESC);
CREATE INDEX idx_realtime_messages_expires ON realtime.messages (expires_at);

-- Outbox for any realtime-originated events (rarely used — mostly for
-- system.notice broadcasts that need to be persisted).
CREATE TABLE IF NOT EXISTS realtime.outbox (
    id                 BIGSERIAL    PRIMARY KEY,
    event_id           UUID         NOT NULL UNIQUE,
    event_type         TEXT         NOT NULL,
    event_version      INTEGER      NOT NULL DEFAULT 1,
    schema_version     INTEGER      NOT NULL DEFAULT 1,
    payload            JSONB        NOT NULL,
    occurred_at        TIMESTAMPTZ  NOT NULL,
    producer           TEXT         NOT NULL,           -- always 'realtime'
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
CREATE INDEX idx_realtime_outbox_pending
    ON realtime.outbox (next_retry_at)
    WHERE published_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS realtime.outbox;
DROP TABLE IF EXISTS realtime.messages;
DROP TABLE IF EXISTS realtime.inbox;
-- +goose StatementEnd
