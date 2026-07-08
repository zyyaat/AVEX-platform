-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS dispatch;

-- Drivers: tracks driver lifecycle and stats
CREATE TABLE dispatch.drivers (
    id                UUID         PRIMARY KEY,
    user_id           UUID         NOT NULL UNIQUE,
    vehicle_type      VARCHAR(20)  NOT NULL,
    license_plate     VARCHAR(50)  NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'offline',
    rating            REAL         NOT NULL DEFAULT 5.0,
    rating_count      INTEGER      NOT NULL DEFAULT 0,
    acceptance_rate   INTEGER      NOT NULL DEFAULT 100,
    completion_rate   INTEGER      NOT NULL DEFAULT 100,
    total_deliveries  INTEGER      NOT NULL DEFAULT 0,
    zone_ids          TEXT[],
    current_order_id  UUID,
    go_online_at      TIMESTAMPTZ,
    go_offline_at     TIMESTAMPTZ,
    suspended_reason  VARCHAR(255),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    version           INTEGER      NOT NULL DEFAULT 1,
    CONSTRAINT chk_driver_status CHECK (status IN ('offline', 'online', 'busy', 'suspended')),
    CONSTRAINT chk_vehicle_type CHECK (vehicle_type IN ('bike', 'scooter', 'car')),
    CONSTRAINT chk_rating_range CHECK (rating >= 0 AND rating <= 5),
    CONSTRAINT chk_acceptance_range CHECK (acceptance_rate >= 0 AND acceptance_rate <= 100),
    CONSTRAINT chk_completion_range CHECK (completion_rate >= 0 AND completion_rate <= 100)
);
CREATE INDEX idx_drivers_user ON dispatch.drivers (user_id);
CREATE INDEX idx_drivers_status ON dispatch.drivers (status) WHERE status = 'online';
CREATE INDEX idx_drivers_online_zone ON dispatch.drivers (status) WHERE status = 'online';

-- Driver locations: latest GPS ping per driver.
-- One row per driver — updated via UPSERT.
-- Stores point as PostGIS geometry for nearest-neighbor queries.
-- NOTE: We use plain (lat, lng) DOUBLE PRECISION columns to avoid a hard
-- dependency on PostGIS. Nearest-neighbor queries use a bounding-box
-- approximation followed by Haversine re-ranking in the service layer.
CREATE TABLE dispatch.driver_locations (
    driver_id     UUID         PRIMARY KEY REFERENCES dispatch.drivers(id) ON DELETE CASCADE,
    lat           DOUBLE PRECISION NOT NULL,
    lng           DOUBLE PRECISION NOT NULL,
    bearing       REAL         NOT NULL DEFAULT 0,
    speed         REAL         NOT NULL DEFAULT 0,
    accuracy      REAL         NOT NULL DEFAULT 0,
    captured_at   TIMESTAMPTZ  NOT NULL,
    received_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_loc_lat CHECK (lat >= -90 AND lat <= 90),
    CONSTRAINT chk_loc_lng CHECK (lng >= -180 AND lng <= 180),
    CONSTRAINT chk_loc_bearing CHECK (bearing >= 0 AND bearing <= 360)
);
CREATE INDEX idx_driver_locations_lat_lng ON dispatch.driver_locations (lat, lng);
CREATE INDEX idx_driver_locations_captured ON dispatch.driver_locations (captured_at);

-- Dispatch offers: tracks each dispatch attempt
CREATE TABLE dispatch.offers (
    id              UUID         PRIMARY KEY,
    order_id        UUID         NOT NULL,
    driver_id       UUID         NOT NULL REFERENCES dispatch.drivers(id) ON DELETE RESTRICT,
    zone_id         VARCHAR(50),
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    pickup_lat      DOUBLE PRECISION NOT NULL,
    pickup_lng      DOUBLE PRECISION NOT NULL,
    delivery_lat    DOUBLE PRECISION NOT NULL,
    delivery_lng    DOUBLE PRECISION NOT NULL,
    est_distance_m  INTEGER,
    est_duration_s  INTEGER,
    est_fare_cents  BIGINT,
    currency        VARCHAR(3)   DEFAULT 'EGP',
    offer_ttl_ms    BIGINT       NOT NULL,
    offered_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ  NOT NULL,
    responded_at    TIMESTAMPTZ,
    accepted_at     TIMESTAMPTZ,
    rejected_at     TIMESTAMPTZ,
    expired_at      TIMESTAMPTZ,
    cancelled_at    TIMESTAMPTZ,
    reject_reason   VARCHAR(255),
    attempt_number  INTEGER      NOT NULL DEFAULT 1,
    created_by      VARCHAR(20)  NOT NULL DEFAULT 'system',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_offer_status CHECK (status IN ('pending', 'accepted', 'rejected', 'expired', 'cancelled')),
    CONSTRAINT chk_offer_pickup_lat CHECK (pickup_lat >= -90 AND pickup_lat <= 90),
    CONSTRAINT chk_offer_pickup_lng CHECK (pickup_lng >= -180 AND pickup_lng <= 180),
    CONSTRAINT chk_offer_delivery_lat CHECK (delivery_lat >= -90 AND delivery_lat <= 90),
    CONSTRAINT chk_offer_delivery_lng CHECK (delivery_lng >= -180 AND delivery_lng <= 180),
    CONSTRAINT chk_offer_attempt_pos CHECK (attempt_number >= 1)
);
CREATE INDEX idx_offers_order ON dispatch.offers (order_id, attempt_number);
CREATE INDEX idx_offers_driver ON dispatch.offers (driver_id, offered_at DESC);
CREATE INDEX idx_offers_pending ON dispatch.offers (expires_at) WHERE status = 'pending';
CREATE INDEX idx_offers_order_pending ON dispatch.offers (order_id) WHERE status = 'pending';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS dispatch.offers;
DROP TABLE IF EXISTS dispatch.driver_locations;
DROP TABLE IF EXISTS dispatch.drivers;
-- +goose StatementEnd
