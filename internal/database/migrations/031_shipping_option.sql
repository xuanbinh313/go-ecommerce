CREATE TABLE IF NOT EXISTS shipping_option (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 price_type TEXT NOT NULL DEFAULT 'flat' CHECK (price_type IN ('flat','calculated')),
 data JSONB,
 metadata JSONB,
 service_zone_id TEXT NOT NULL REFERENCES service_zone(id) ON UPDATE CASCADE ON DELETE CASCADE,
 shipping_profile_id TEXT REFERENCES shipping_profile(id) ON UPDATE CASCADE ON DELETE SET NULL,
 provider_id TEXT REFERENCES fulfillment_provider(id) ON UPDATE CASCADE ON DELETE SET NULL,
 shipping_option_type_id TEXT NOT NULL REFERENCES shipping_option_type(id) ON UPDATE CASCADE ON DELETE CASCADE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_shipping_option_service_zone_id ON shipping_option(service_zone_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_shipping_option_shipping_profile_id ON shipping_option(shipping_profile_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_shipping_option_provider_id ON shipping_option(provider_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_shipping_option_shipping_option_type_id ON shipping_option(shipping_option_type_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS shipping_option_rule (
 id TEXT PRIMARY KEY,
 attribute TEXT NOT NULL,
 operator TEXT NOT NULL CHECK (operator IN ('in','eq','ne','gt','gte','lt','lte','nin')),
 value JSONB,
 shipping_option_id TEXT NOT NULL REFERENCES shipping_option(id) ON UPDATE CASCADE ON DELETE CASCADE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_shipping_option_rule_shipping_option_id ON shipping_option_rule(shipping_option_id) WHERE deleted_at IS NULL;
