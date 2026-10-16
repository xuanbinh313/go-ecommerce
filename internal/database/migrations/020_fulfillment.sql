CREATE TABLE IF NOT EXISTS fulfillment_provider (
 id TEXT PRIMARY KEY, is_enabled BOOLEAN NOT NULL DEFAULT true,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS fulfillment_set (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS service_zone (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, fulfillment_set_id TEXT NOT NULL REFERENCES fulfillment_set(id) ON UPDATE CASCADE ON DELETE CASCADE, metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_service_zone_name ON service_zone(name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_service_zone_fulfillment_set ON service_zone(fulfillment_set_id) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS geo_zone (
 id TEXT PRIMARY KEY, type TEXT NOT NULL, country_code TEXT NOT NULL, province_code TEXT, city TEXT, postal_expression JSONB,
 service_zone_id TEXT NOT NULL REFERENCES service_zone(id) ON UPDATE CASCADE ON DELETE CASCADE, metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_geo_zone_service_zone ON geo_zone(service_zone_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_geo_zone_country_code ON geo_zone(country_code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_geo_zone_province_code ON geo_zone(province_code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_geo_zone_city ON geo_zone(city) WHERE deleted_at IS NULL;
