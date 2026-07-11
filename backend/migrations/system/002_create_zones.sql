-- ===== Zones table =====
-- Stores delivery zones as polygons (GeoJSON) + center point + radius.
-- Used by: catalog.restaurants.zone_id, dispatch.drivers.zone_ids,
-- financial.pricing_rules.zone_id, financial.surge_zones.zone_id.

-- +goose Up
CREATE TABLE IF NOT EXISTS system.zones (
    id              VARCHAR(50)   PRIMARY KEY,
    name            VARCHAR(255)  NOT NULL,
    name_ar         VARCHAR(255),
    center_lat      DOUBLE PRECISION NOT NULL,
    center_lng      DOUBLE PRECISION NOT NULL,
    radius_m        INTEGER NOT NULL DEFAULT 3000,
    polygon_geojson TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_zones_active ON system.zones (is_active) WHERE is_active = TRUE;

-- +goose Down
DROP TABLE IF EXISTS system.zones;
