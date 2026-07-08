-- +goose Up
-- +goose StatementBegin
-- Dispatch module: outbox table (transactional event outbox).
CREATE TABLE IF NOT EXISTS dispatch.outbox (
    id                 BIGSERIAL    PRIMARY KEY,
    event_id           UUID         NOT NULL UNIQUE,
    event_type         TEXT         NOT NULL,
    event_version      INTEGER      NOT NULL DEFAULT 1,
    schema_version     INTEGER      NOT NULL DEFAULT 1,
    payload            JSONB        NOT NULL,
    occurred_at        TIMESTAMPTZ  NOT NULL,
    producer           TEXT         NOT NULL,           -- always 'dispatch'
    correlation_id     TEXT,
    trace_id           TEXT,
    actor_type         TEXT,
    actor_id           TEXT,
    actor_ip           TEXT,
    actor_user_agent   TEXT,
    published_at       TIMESTAMPTZ,                     -- NULL = not yet published
    retry_count        INTEGER      NOT NULL DEFAULT 0,
    next_retry_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_error         TEXT,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dispatch_outbox_pending
    ON dispatch.outbox (next_retry_at)
    WHERE published_at IS NULL;

CREATE INDEX idx_dispatch_outbox_published_at
    ON dispatch.outbox (published_at)
    WHERE published_at IS NOT NULL;

-- Dispatch module: inbox table (consumer-side idempotency).
-- Used to deduplicate events consumed from the orders module.
CREATE TABLE IF NOT EXISTS dispatch.inbox (
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
DROP TABLE IF EXISTS dispatch.inbox;
DROP TABLE IF EXISTS dispatch.outbox;
-- +goose StatementEnd
