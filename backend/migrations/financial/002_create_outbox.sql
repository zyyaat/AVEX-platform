-- +goose Up
-- +goose StatementBegin
-- Financial module: outbox table (transactional event outbox).
--
-- Mirrors the orders.outbox schema so the same outbox worker code
-- (cmd/worker + platform/outbox) can process both modules uniformly.
CREATE TABLE IF NOT EXISTS financial.outbox (
    id                 BIGSERIAL    PRIMARY KEY,
    event_id           UUID         NOT NULL UNIQUE,
    event_type         TEXT         NOT NULL,
    event_version      INTEGER      NOT NULL DEFAULT 1,
    schema_version     INTEGER      NOT NULL DEFAULT 1,
    payload            JSONB        NOT NULL,
    occurred_at        TIMESTAMPTZ  NOT NULL,
    producer           TEXT         NOT NULL,           -- always 'financial'
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

CREATE INDEX idx_financial_outbox_pending
    ON financial.outbox (next_retry_at)
    WHERE published_at IS NULL;

CREATE INDEX idx_financial_outbox_published_at
    ON financial.outbox (published_at)
    WHERE published_at IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS financial.outbox;
-- +goose StatementEnd
