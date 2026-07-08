-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS support;

CREATE TABLE support.tickets (
    id                UUID         PRIMARY KEY,
    ticket_no         VARCHAR(50)  NOT NULL UNIQUE,
    user_id           UUID         NOT NULL,
    order_id          UUID,
    driver_id         UUID,
    restaurant_id     UUID,
    subject           VARCHAR(255) NOT NULL,
    description       TEXT         NOT NULL,
    category          VARCHAR(50)  NOT NULL,
    priority          VARCHAR(20)  NOT NULL DEFAULT 'normal',
    status            VARCHAR(20)  NOT NULL DEFAULT 'open',
    assigned_to       UUID,
    created_by        VARCHAR(20)  NOT NULL DEFAULT 'user',
    closed_by         VARCHAR(20),
    closed_reason     TEXT,
    message_count     INTEGER      NOT NULL DEFAULT 0,
    first_response_at TIMESTAMPTZ,
    resolved_at       TIMESTAMPTZ,
    closed_at         TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    version           INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT chk_ticket_status CHECK (status IN ('open', 'in_progress', 'waiting', 'resolved', 'closed')),
    CONSTRAINT chk_ticket_priority CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT chk_ticket_category CHECK (category IN ('order_issue', 'payment_issue', 'delivery_issue', 'account_issue', 'app_bug', 'feature_request', 'other'))
);
CREATE INDEX idx_tickets_user ON support.tickets (user_id, created_at DESC);
CREATE INDEX idx_tickets_agent ON support.tickets (assigned_to, status) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_tickets_status ON support.tickets (status);
CREATE INDEX idx_tickets_unassigned ON support.tickets (created_at) WHERE assigned_to IS NULL AND status = 'open';

CREATE TABLE support.ticket_messages (
    id           UUID         PRIMARY KEY,
    ticket_id    UUID         NOT NULL REFERENCES support.tickets(id) ON DELETE CASCADE,
    sender_type  VARCHAR(20)  NOT NULL,
    sender_id    UUID         NOT NULL,
    body         TEXT         NOT NULL,
    edited_at    TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_msg_sender_type CHECK (sender_type IN ('user', 'agent', 'system', 'internal'))
);
CREATE INDEX idx_messages_ticket ON support.ticket_messages (ticket_id, created_at ASC);

CREATE TABLE support.ticket_attachments (
    id          UUID         PRIMARY KEY,
    message_id  UUID         NOT NULL REFERENCES support.ticket_messages(id) ON DELETE CASCADE,
    file_name   VARCHAR(255) NOT NULL,
    file_type   VARCHAR(20)  NOT NULL,
    file_url    TEXT         NOT NULL,
    file_size   BIGINT       NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_attach_file_type CHECK (file_type IN ('image', 'document', 'video', 'audio')),
    CONSTRAINT chk_attach_file_size CHECK (file_size > 0)
);
CREATE INDEX idx_attachments_message ON support.ticket_attachments (message_id);

CREATE TABLE IF NOT EXISTS support.outbox (
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
CREATE INDEX idx_support_outbox_pending ON support.outbox (next_retry_at) WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS support.inbox (
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
DROP TABLE IF EXISTS support.inbox;
DROP TABLE IF EXISTS support.outbox;
DROP TABLE IF EXISTS support.ticket_attachments;
DROP TABLE IF EXISTS support.ticket_messages;
DROP TABLE IF EXISTS support.tickets;
-- +goose StatementEnd
