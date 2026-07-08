-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS permissions;

CREATE TABLE permissions.roles (
    id          UUID         PRIMARY KEY,
    name        VARCHAR(50)  NOT NULL UNIQUE,
    description TEXT,
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE permissions.permissions (
    id          UUID         PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    module      VARCHAR(50)  NOT NULL,
    resource    VARCHAR(50)  NOT NULL,
    action      VARCHAR(50)  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_perm_name CHECK (name ~ '^[a-z_*]+\.[a-z_*]+\.[a-z_*]+$')
);
CREATE INDEX idx_permissions_module ON permissions.permissions (module);

CREATE TABLE permissions.role_permissions (
    id            UUID         PRIMARY KEY,
    role_id       UUID         NOT NULL REFERENCES permissions.roles(id) ON DELETE CASCADE,
    permission_id UUID         NOT NULL REFERENCES permissions.permissions(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(role_id, permission_id)
);
CREATE INDEX idx_role_permissions_role ON permissions.role_permissions (role_id);

CREATE TABLE permissions.user_roles (
    id          UUID         PRIMARY KEY,
    user_id     UUID         NOT NULL,
    role_id     UUID         NOT NULL REFERENCES permissions.roles(id) ON DELETE CASCADE,
    assigned_by UUID,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id)
);
CREATE INDEX idx_user_roles_user ON permissions.user_roles (user_id);
CREATE INDEX idx_user_roles_role ON permissions.user_roles (role_id);

CREATE TABLE IF NOT EXISTS permissions.outbox (
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
CREATE INDEX idx_permissions_outbox_pending ON permissions.outbox (next_retry_at) WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS permissions.inbox (
    event_id      UUID         NOT NULL,
    handler_name  VARCHAR(100) NOT NULL,
    event_type    TEXT,
    processed_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, handler_name)
);

-- Seed system roles
INSERT INTO permissions.roles (id, name, description, is_system) VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin', 'System Administrator', TRUE),
    ('00000000-0000-0000-0000-000000000002', 'agent', 'Support Agent', TRUE),
    ('00000000-0000-0000-0000-000000000003', 'merchant', 'Restaurant Merchant', TRUE),
    ('00000000-0000-0000-0000-000000000004', 'driver', 'Delivery Driver', TRUE),
    ('00000000-0000-0000-0000-000000000005', 'user', 'Regular Customer', TRUE)
ON CONFLICT (name) DO NOTHING;

-- Seed core permissions
INSERT INTO permissions.permissions (id, name, description, module, resource, action) VALUES
    ('00000000-0000-0000-0001-000000000001', 'orders.order.read',    'Read orders',         'orders', 'order', 'read'),
    ('00000000-0000-0000-0001-000000000002', 'orders.order.write',   'Create/update orders','orders', 'order', 'write'),
    ('00000000-0000-0000-0001-000000000003', 'orders.order.cancel',  'Cancel orders',       'orders', 'order', 'cancel'),
    ('00000000-0000-0000-0002-000000000001', 'catalog.restaurant.read',  'Read restaurants',  'catalog', 'restaurant', 'read'),
    ('00000000-0000-0000-0002-000000000002', 'catalog.restaurant.write', 'Manage restaurants','catalog', 'restaurant', 'write'),
    ('00000000-0000-0000-0003-000000000001', 'financial.wallet.read',   'Read wallet',       'financial', 'wallet', 'read'),
    ('00000000-0000-0000-0003-000000000002', 'financial.wallet.credit', 'Credit wallet',     'financial', 'wallet', 'credit'),
    ('00000000-0000-0000-0003-000000000003', 'financial.wallet.debit',  'Debit wallet',      'financial', 'wallet', 'debit'),
    ('00000000-0000-0000-0004-000000000001', 'dispatch.driver.read',   'Read driver info',   'dispatch', 'driver', 'read'),
    ('00000000-0000-0000-0004-000000000002', 'dispatch.driver.write',  'Manage drivers',     'dispatch', 'driver', 'write'),
    ('00000000-0000-0000-0005-000000000001', 'support.ticket.read',    'Read tickets',       'support', 'ticket', 'read'),
    ('00000000-0000-0000-0005-000000000002', 'support.ticket.write',   'Create/update tickets','support','ticket','write'),
    ('00000000-0000-0000-0005-000000000003', 'support.ticket.assign',  'Assign tickets',     'support', 'ticket', 'assign'),
    ('00000000-0000-0000-0006-000000000001', 'permissions.role.read',  'Read roles',         'permissions', 'role', 'read'),
    ('00000000-0000-0000-0006-000000000002', 'permissions.role.write', 'Manage roles',       'permissions', 'role', 'write'),
    ('00000000-0000-0000-0006-000000000003', 'permissions.*',          'Wildcard admin perms','permissions', '*', '*')
ON CONFLICT (name) DO NOTHING;

-- Grant admin role all permissions (wildcard)
INSERT INTO permissions.role_permissions (id, role_id, permission_id) VALUES
    ('00000000-0000-0000-0007-000000000001', '00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0006-000000000003')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Grant user role basic permissions
INSERT INTO permissions.role_permissions (id, role_id, permission_id) VALUES
    ('00000000-0000-0000-0008-000000000001', '00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0001-000000000001'),
    ('00000000-0000-0000-0008-000000000002', '00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0001-000000000002'),
    ('00000000-0000-0000-0008-000000000003', '00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0002-000000000001'),
    ('00000000-0000-0000-0008-000000000004', '00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0003-000000000001'),
    ('00000000-0000-0000-0008-000000000005', '00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0005-000000000001'),
    ('00000000-0000-0000-0008-000000000006', '00000000-0000-0000-0000-000000000005', '00000000-0000-0000-0005-000000000002')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Grant driver role dispatch permissions
INSERT INTO permissions.role_permissions (id, role_id, permission_id) VALUES
    ('00000000-0000-0000-0009-000000000001', '00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0004-000000000001'),
    ('00000000-0000-0000-0009-000000000002', '00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0003-000000000001'),
    ('00000000-0000-0000-0009-000000000003', '00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0001-000000000001')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Grant merchant role catalog permissions
INSERT INTO permissions.role_permissions (id, role_id, permission_id) VALUES
    ('00000000-0000-0000-000a-000000000001', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0002-000000000001'),
    ('00000000-0000-0000-000a-000000000002', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0002-000000000002'),
    ('00000000-0000-0000-000a-000000000003', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0001-000000000001'),
    ('00000000-0000-0000-000a-000000000004', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0005-000000000001'),
    ('00000000-0000-0000-000a-000000000005', '00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0005-000000000002')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Grant agent role support permissions
INSERT INTO permissions.role_permissions (id, role_id, permission_id) VALUES
    ('00000000-0000-0000-000b-000000000001', '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0005-000000000001'),
    ('00000000-0000-0000-000b-000000000002', '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0005-000000000002'),
    ('00000000-0000-0000-000b-000000000003', '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0005-000000000003'),
    ('00000000-0000-0000-000b-000000000004', '00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0001-000000000001')
ON CONFLICT (role_id, permission_id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS permissions.inbox;
DROP TABLE IF EXISTS permissions.outbox;
DROP TABLE IF EXISTS permissions.user_roles;
DROP TABLE IF EXISTS permissions.role_permissions;
DROP TABLE IF EXISTS permissions.permissions;
DROP TABLE IF EXISTS permissions.roles;
-- +goose StatementEnd
