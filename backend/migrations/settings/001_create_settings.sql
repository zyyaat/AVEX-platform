-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS settings;

CREATE TABLE settings.settings (
    id           UUID         PRIMARY KEY,
    key          VARCHAR(100) NOT NULL UNIQUE,
    description  TEXT,
    setting_type VARCHAR(20)  NOT NULL,
    value        TEXT         NOT NULL,
    is_protected BOOLEAN      NOT NULL DEFAULT FALSE,
    version      INTEGER      NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_setting_type CHECK (setting_type IN ('string', 'int', 'float', 'bool', 'json'))
);

CREATE TABLE settings.setting_revisions (
    id          UUID         PRIMARY KEY,
    setting_id  UUID         NOT NULL REFERENCES settings.settings(id) ON DELETE CASCADE,
    version     INTEGER      NOT NULL,
    value       TEXT         NOT NULL,
    changed_by  VARCHAR(100),
    change_note TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(setting_id, version)
);
CREATE INDEX idx_revisions_setting ON settings.setting_revisions (setting_id, version DESC);

CREATE TABLE settings.feature_flags (
    id           UUID         PRIMARY KEY,
    name         VARCHAR(100) NOT NULL UNIQUE,
    description  TEXT,
    enabled      BOOLEAN      NOT NULL DEFAULT FALSE,
    target_type  VARCHAR(20)  NOT NULL DEFAULT 'all',
    target_value TEXT,
    rollout_pct  INTEGER      NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_flag_target CHECK (target_type IN ('all', 'users', 'roles', 'percent')),
    CONSTRAINT chk_flag_rollout CHECK (rollout_pct >= 0 AND rollout_pct <= 100)
);
CREATE INDEX idx_feature_flags_enabled ON settings.feature_flags (enabled) WHERE enabled = TRUE;

CREATE TABLE IF NOT EXISTS settings.outbox (
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
CREATE INDEX idx_settings_outbox_pending ON settings.outbox (next_retry_at) WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS settings.inbox (
    event_id      UUID         NOT NULL,
    handler_name  VARCHAR(100) NOT NULL,
    event_type    TEXT,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, handler_name)
);

-- Seed default settings
INSERT INTO settings.settings (id, key, description, setting_type, value, is_protected) VALUES
    ('00000000-0000-0000-0001-000000000001', 'app.name',              'Application name',           'string', 'AVEX Delivery', TRUE),
    ('00000000-0000-0000-0001-000000000002', 'app.maintenance_mode',  'Global maintenance mode',   'bool',   'false',         TRUE),
    ('00000000-0000-0000-0001-000000000003', 'delivery.default_radius_km', 'Default driver search radius (km)', 'int', '5', FALSE),
    ('00000000-0000-0000-0001-000000000004', 'delivery.max_attempts', 'Max dispatch attempts per order', 'int', '5', FALSE),
    ('00000000-0000-0000-0001-000000000005', 'delivery.offer_ttl_seconds', 'Driver offer TTL (seconds)', 'int', '15', FALSE),
    ('00000000-0000-0000-0001-000000000006', 'financial.currency',    'Default currency',           'string', 'EGP',           FALSE),
    ('00000000-0000-0000-0001-000000000007', 'financial.vat_rate',    'VAT rate (%)',               'float',  '14.0',          FALSE),
    ('00000000-0000-0000-0001-000000000008', 'notifications.max_retries', 'Max notification retries', 'int', '3', FALSE)
ON CONFLICT (key) DO NOTHING;

-- Seed initial revisions
INSERT INTO settings.setting_revisions (id, setting_id, version, value, changed_by, change_note)
SELECT gen_random_uuid(), id, 1, value, 'system', 'Initial value'
FROM settings.settings
ON CONFLICT DO NOTHING;

-- Seed default feature flags
INSERT INTO settings.feature_flags (id, name, description, enabled, target_type, rollout_pct) VALUES
    ('00000000-0000-0000-0002-000000000001', 'new_checkout_ui',   'New checkout UI',       FALSE, 'percent', 0),
    ('00000000-0000-0000-0002-000000000002', 'live_order_tracking','Live order tracking',  TRUE,  'all',     0),
    ('00000000-0000-0000-0002-000000000003', 'wallet_payments',   'Wallet payment option', TRUE,  'all',     0),
    ('00000000-0000-0000-0002-000000000004', 'beta_features',     'Beta features',         FALSE, 'roles',   0)
ON CONFLICT (name) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS settings.inbox;
DROP TABLE IF EXISTS settings.outbox;
DROP TABLE IF EXISTS settings.feature_flags;
DROP TABLE IF EXISTS settings.setting_revisions;
DROP TABLE IF EXISTS settings.settings;
-- +goose StatementEnd
