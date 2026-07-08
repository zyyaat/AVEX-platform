-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS financial;

-- Wallets: one per (owner_type, owner_id, currency)
CREATE TABLE financial.wallets (
    id              UUID         PRIMARY KEY,
    owner_type      VARCHAR(20)  NOT NULL,
    owner_id        UUID         NOT NULL,
    currency        VARCHAR(3)   NOT NULL,
    balance         BIGINT       NOT NULL DEFAULT 0,
    pending_balance BIGINT       NOT NULL DEFAULT 0,
    status          VARCHAR(20)  NOT NULL DEFAULT 'active',
    version         INTEGER      NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(owner_type, owner_id, currency),
    CONSTRAINT chk_wallet_owner CHECK (owner_type IN ('user', 'driver', 'merchant')),
    CONSTRAINT chk_wallet_status CHECK (status IN ('active', 'frozen', 'closed')),
    CONSTRAINT chk_wallet_balance_nonneg CHECK (balance >= 0),
    CONSTRAINT chk_wallet_pending_nonneg CHECK (pending_balance >= 0)
);
CREATE UNIQUE INDEX idx_wallets_owner ON financial.wallets (owner_type, owner_id, currency);
CREATE INDEX idx_wallets_owner_id ON financial.wallets (owner_id);

-- Transactions: append-only ledger
CREATE TABLE financial.transactions (
    id               UUID         PRIMARY KEY,
    wallet_id        UUID         NOT NULL REFERENCES financial.wallets(id) ON DELETE RESTRICT,
    type             VARCHAR(20)  NOT NULL,
    category         VARCHAR(50)  NOT NULL,
    amount           BIGINT       NOT NULL,
    currency         VARCHAR(3)   NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'pending',
    reference_type   VARCHAR(50),
    reference_id     VARCHAR(100),
    description      TEXT,
    metadata         JSONB,
    idempotency_key  VARCHAR(100) UNIQUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    completed_at     TIMESTAMPTZ,
    CONSTRAINT chk_txn_type CHECK (type IN ('credit', 'debit')),
    CONSTRAINT chk_txn_status CHECK (status IN ('pending', 'completed', 'failed', 'reversed')),
    CONSTRAINT chk_txn_amount_pos CHECK (amount > 0),
    CONSTRAINT chk_txn_category CHECK (category IN (
        'topup', 'order_payment', 'refund', 'payout',
        'commission', 'tip', 'adjustment', 'promotion'
    ))
);
CREATE INDEX idx_txn_wallet ON financial.transactions (wallet_id, created_at DESC);
CREATE INDEX idx_txn_reference ON financial.transactions (reference_type, reference_id) WHERE reference_id IS NOT NULL;

-- Pricing rules per zone
CREATE TABLE financial.pricing_rules (
    id                       UUID         PRIMARY KEY,
    zone_id                  VARCHAR(50)  NOT NULL,
    currency                 VARCHAR(3)   NOT NULL DEFAULT 'EGP',
    base_fee                 BIGINT       NOT NULL,
    per_km_rate              BIGINT       NOT NULL DEFAULT 0,
    per_min_rate             BIGINT       NOT NULL DEFAULT 0,
    min_fee                  BIGINT       NOT NULL DEFAULT 0,
    max_fee                  BIGINT,
    free_delivery_threshold  BIGINT,
    is_active                BOOLEAN      NOT NULL DEFAULT TRUE,
    valid_from               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_pricing_base_nonneg CHECK (base_fee >= 0),
    CONSTRAINT chk_pricing_perkm_nonneg CHECK (per_km_rate >= 0),
    CONSTRAINT chk_pricing_permin_nonneg CHECK (per_min_rate >= 0),
    CONSTRAINT chk_pricing_min_nonneg CHECK (min_fee >= 0),
    CONSTRAINT chk_pricing_max_nonneg CHECK (max_fee IS NULL OR max_fee >= 0),
    CONSTRAINT chk_pricing_threshold_nonneg CHECK (free_delivery_threshold IS NULL OR free_delivery_threshold >= 0)
);
CREATE INDEX idx_pricing_zone_active ON financial.pricing_rules (zone_id, currency, is_active);
CREATE INDEX idx_pricing_valid_window ON financial.pricing_rules (valid_from, valid_to) WHERE is_active = TRUE;

-- Surge zones
CREATE TABLE financial.surge_zones (
    id           UUID         PRIMARY KEY,
    zone_id      VARCHAR(50)  NOT NULL,
    multiplier   REAL         NOT NULL,
    reason       VARCHAR(100),
    day_of_week  INTEGER,
    start_time   VARCHAR(10)  DEFAULT '00:00',
    end_time     VARCHAR(10)  DEFAULT '23:59',
    is_active    BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_surge_multiplier CHECK (multiplier >= 1.0),
    CONSTRAINT chk_surge_dow CHECK (day_of_week IS NULL OR (day_of_week >= 0 AND day_of_week <= 6))
);
CREATE INDEX idx_surge_zone_active ON financial.surge_zones (zone_id, is_active);

-- Promotions
CREATE TABLE financial.promotions (
    id                    UUID         PRIMARY KEY,
    code                  VARCHAR(50)  NOT NULL UNIQUE,
    description           TEXT,
    promo_type            VARCHAR(20)  NOT NULL,
    value                 BIGINT       NOT NULL,
    currency              VARCHAR(3)   NOT NULL DEFAULT 'EGP',
    min_order_amount      BIGINT       NOT NULL DEFAULT 0,
    max_discount_amount   BIGINT,
    usage_limit           INTEGER,
    usage_count           INTEGER      NOT NULL DEFAULT 0,
    per_user_limit        INTEGER      NOT NULL DEFAULT 1,
    valid_from            TIMESTAMPTZ  NOT NULL,
    valid_to              TIMESTAMPTZ,
    is_active             BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_promo_type CHECK (promo_type IN ('percent', 'fixed', 'free_delivery')),
    CONSTRAINT chk_promo_value_nonneg CHECK (value >= 0),
    CONSTRAINT chk_promo_percent CHECK (promo_type != 'percent' OR value <= 100),
    CONSTRAINT chk_promo_min_nonneg CHECK (min_order_amount >= 0),
    CONSTRAINT chk_promo_max_nonneg CHECK (max_discount_amount IS NULL OR max_discount_amount >= 0),
    CONSTRAINT chk_promo_per_user_pos CHECK (per_user_limit >= 1)
);
CREATE INDEX idx_promo_active ON financial.promotions (is_active, valid_from, valid_to);

-- Promotion redemptions
CREATE TABLE financial.promotion_redemptions (
    id              UUID         PRIMARY KEY,
    promotion_id    UUID         NOT NULL REFERENCES financial.promotions(id) ON DELETE RESTRICT,
    user_id         UUID         NOT NULL,
    order_id        UUID,
    discount_amount BIGINT       NOT NULL,
    currency        VARCHAR(3)   NOT NULL,
    redeemed_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(promotion_id, user_id, order_id),
    CONSTRAINT chk_redemption_discount_nonneg CHECK (discount_amount >= 0)
);
CREATE INDEX idx_redemption_user ON financial.promotion_redemptions (user_id, redeemed_at DESC);
CREATE INDEX idx_redemption_promo ON financial.promotion_redemptions (promotion_id, user_id);
CREATE INDEX idx_redemption_order ON financial.promotion_redemptions (order_id) WHERE order_id IS NOT NULL;

-- Taxes
CREATE TABLE financial.taxes (
    id          UUID         PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    rate        REAL         NOT NULL,
    applies_to  VARCHAR(50)  NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_tax_rate_nonneg CHECK (rate >= 0),
    CONSTRAINT chk_tax_applies_to CHECK (applies_to IN ('delivery_fee', 'order_total', 'service_fee'))
);
CREATE INDEX idx_tax_active ON financial.taxes (is_active, applies_to);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS financial.taxes;
DROP TABLE IF EXISTS financial.promotion_redemptions;
DROP TABLE IF EXISTS financial.promotions;
DROP TABLE IF EXISTS financial.surge_zones;
DROP TABLE IF EXISTS financial.pricing_rules;
DROP TABLE IF EXISTS financial.transactions;
DROP TABLE IF EXISTS financial.wallets;
-- +goose StatementEnd
