CREATE TABLE IF NOT EXISTS fulfillment_address (
 id TEXT PRIMARY KEY, company TEXT, first_name TEXT, last_name TEXT, address_1 TEXT, address_2 TEXT,
 city TEXT, country_code TEXT, province TEXT, postal_code TEXT, phone TEXT, metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS fulfillment (
 id TEXT PRIMARY KEY, location_id TEXT NOT NULL, packed_at TIMESTAMPTZ, shipped_at TIMESTAMPTZ, marked_shipped_by TEXT,
 created_by TEXT, delivered_at TIMESTAMPTZ, canceled_at TIMESTAMPTZ, data JSONB, requires_shipping BOOLEAN NOT NULL DEFAULT true,
 provider_id TEXT REFERENCES fulfillment_provider(id) ON UPDATE CASCADE ON DELETE SET NULL,
 shipping_option_id TEXT REFERENCES shipping_option(id) ON UPDATE CASCADE ON DELETE SET NULL, metadata JSONB,
 delivery_address_id TEXT REFERENCES fulfillment_address(id) ON UPDATE CASCADE ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_fulfillment_location_id ON fulfillment(location_id) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS fulfillment_item (
 id TEXT PRIMARY KEY, title TEXT NOT NULL, sku TEXT NOT NULL, barcode TEXT NOT NULL, quantity NUMERIC NOT NULL,
 raw_quantity JSONB NOT NULL, line_item_id TEXT, inventory_item_id TEXT, fulfillment_id TEXT NOT NULL REFERENCES fulfillment(id) ON UPDATE CASCADE ON DELETE CASCADE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_fulfillment_item_fulfillment_id ON fulfillment_item(fulfillment_id) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS fulfillment_label (
 id TEXT PRIMARY KEY, tracking_number TEXT NOT NULL, tracking_url TEXT NOT NULL, label_url TEXT NOT NULL,
 fulfillment_id TEXT NOT NULL REFERENCES fulfillment(id) ON UPDATE CASCADE ON DELETE CASCADE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_fulfillment_label_fulfillment_id ON fulfillment_label(fulfillment_id) WHERE deleted_at IS NULL;
