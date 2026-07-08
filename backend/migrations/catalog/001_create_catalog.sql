-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS catalog;

CREATE TABLE catalog.restaurants (
    id                UUID          PRIMARY KEY,
    name              VARCHAR(255)  NOT NULL,
    name_ar           VARCHAR(255),
    description       TEXT,
    description_ar    TEXT,
    image_url         TEXT,
    cover_url         TEXT,
    cuisines          TEXT,
    lat               DOUBLE PRECISION NOT NULL,
    lng               DOUBLE PRECISION NOT NULL,
    zone_id           TEXT,
    merchant_id       UUID,
    rating            REAL DEFAULT 4.5,
    rating_count      INTEGER DEFAULT 0,
    delivery_time_min INTEGER DEFAULT 20,
    delivery_time_max INTEGER DEFAULT 45,
    delivery_fee      REAL DEFAULT 3.99,
    min_order         REAL DEFAULT 0,
    is_active         BOOLEAN DEFAULT TRUE,
    is_pro            BOOLEAN DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_restaurants_active ON catalog.restaurants (is_active) WHERE is_active = TRUE;
CREATE INDEX idx_restaurants_zone ON catalog.restaurants (zone_id) WHERE is_active = TRUE;
CREATE INDEX idx_restaurants_merchant ON catalog.restaurants (merchant_id);
CREATE INDEX idx_restaurants_pro ON catalog.restaurants (is_pro) WHERE is_pro = TRUE AND is_active = TRUE;

CREATE TABLE catalog.categories (
    id          UUID         PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    name_ar     VARCHAR(255),
    icon        VARCHAR(50) DEFAULT '🍽️',
    image_url   TEXT,
    sort_order  INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_categories_sort ON catalog.categories (sort_order);

CREATE TABLE catalog.menu_items (
    id              UUID          PRIMARY KEY,
    restaurant_id   UUID          NOT NULL REFERENCES catalog.restaurants(id) ON DELETE CASCADE,
    category_id     UUID,
    name            VARCHAR(255)  NOT NULL,
    name_ar         VARCHAR(255),
    description     TEXT,
    description_ar  TEXT,
    price           REAL          NOT NULL,
    image           VARCHAR(50) DEFAULT '🍽️',
    image_url       TEXT,
    is_popular      BOOLEAN DEFAULT FALSE,
    is_available    BOOLEAN DEFAULT TRUE,
    rating          REAL DEFAULT 4.5,
    rating_count    INTEGER DEFAULT 0,
    prep_time       INTEGER DEFAULT 15,
    calories        INTEGER DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_price_positive CHECK (price >= 0)
);
CREATE INDEX idx_menu_items_restaurant ON catalog.menu_items (restaurant_id, is_available);
CREATE INDEX idx_menu_items_category ON catalog.menu_items (category_id, is_available);
CREATE INDEX idx_menu_items_popular ON catalog.menu_items (is_popular, is_available) WHERE is_popular = TRUE AND is_available = TRUE;

CREATE TABLE catalog.store_hours (
    id            UUID         PRIMARY KEY,
    restaurant_id UUID         NOT NULL REFERENCES catalog.restaurants(id) ON DELETE CASCADE,
    day_of_week   INTEGER      NOT NULL,
    open_time     VARCHAR(10)  DEFAULT '10:00',
    close_time    VARCHAR(10)  DEFAULT '23:00',
    is_open       BOOLEAN      DEFAULT TRUE,
    UNIQUE(restaurant_id, day_of_week),
    CONSTRAINT chk_day_of_week CHECK (day_of_week >= 0 AND day_of_week <= 6)
);
CREATE INDEX idx_store_hours_restaurant ON catalog.store_hours (restaurant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS catalog.store_hours;
DROP TABLE IF EXISTS catalog.menu_items;
DROP TABLE IF EXISTS catalog.categories;
DROP TABLE IF EXISTS catalog.restaurants;
-- +goose StatementEnd
