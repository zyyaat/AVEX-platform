-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS audit;

-- Audit entries: append-only, never updated or deleted
CREATE TABLE audit.entries (
    id             UUID         PRIMARY KEY,
    actor_type     VARCHAR(20)  NOT NULL,
    actor_id       VARCHAR(100),
    action         VARCHAR(100) NOT NULL,
    resource_type  VARCHAR(50)  NOT NULL,
    resource_id    VARCHAR(100),
    severity       VARCHAR(20)  NOT NULL DEFAULT 'info',
    description    TEXT,
    metadata       JSONB,
    ip_address     VARCHAR(45),
    user_agent     TEXT,
    correlation_id VARCHAR(100),
    trace_id       VARCHAR(100),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_audit_actor CHECK (actor_type IN ('user', 'driver', 'merchant', 'agent', 'admin', 'system')),
    CONSTRAINT chk_audit_severity CHECK (severity IN ('info', 'warning', 'critical')),
    CONSTRAINT chk_audit_actor_id CHECK (actor_id IS NOT NULL OR actor_type = 'system')
);
-- No UPDATE or DELETE permissions granted — enforced at application level.
-- Indexes for common query patterns
CREATE INDEX idx_audit_actor ON audit.entries (actor_type, actor_id, created_at DESC);
CREATE INDEX idx_audit_resource ON audit.entries (resource_type, resource_id, created_at DESC);
CREATE INDEX idx_audit_action ON audit.entries (action, created_at DESC);
CREATE INDEX idx_audit_severity ON audit.entries (severity, created_at DESC) WHERE severity IN ('warning', 'critical');
CREATE INDEX idx_audit_created ON audit.entries (created_at DESC);
CREATE INDEX idx_audit_correlation ON audit.entries (correlation_id) WHERE correlation_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS audit.outbox (
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
CREATE INDEX idx_audit_outbox_pending ON audit.outbox (next_retry_at) WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS audit.inbox (
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
DROP TABLE IF EXISTS audit.inbox;
DROP TABLE IF EXISTS audit.outbox;
DROP TABLE IF EXISTS audit.entries;
-- +goose StatementEnd
