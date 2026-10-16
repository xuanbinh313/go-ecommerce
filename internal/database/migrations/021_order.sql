CREATE TABLE IF NOT EXISTS order_address (
 id TEXT PRIMARY KEY, customer_id TEXT, company TEXT, first_name TEXT, last_name TEXT, address_1 TEXT, address_2 TEXT,
 city TEXT, country_code TEXT, province TEXT, postal_code TEXT, phone TEXT, metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_address_customer_id ON order_address(customer_id);
CREATE TABLE IF NOT EXISTS "order" (
 id TEXT PRIMARY KEY, region_id TEXT, display_id BIGSERIAL, custom_display_id TEXT, customer_id TEXT, version INTEGER NOT NULL DEFAULT 1,
 sales_channel_id TEXT, status TEXT NOT NULL DEFAULT 'pending', is_draft_order BOOLEAN NOT NULL DEFAULT false, email TEXT,
 currency_code TEXT NOT NULL, locale TEXT, no_notification BOOLEAN, metadata JSONB, canceled_at TIMESTAMPTZ,
 shipping_address_id TEXT REFERENCES order_address(id) ON UPDATE CASCADE ON DELETE SET NULL,
 billing_address_id TEXT REFERENCES order_address(id) ON UPDATE CASCADE ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_display_id ON "order"(display_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_order_custom_display_id ON "order"(custom_display_id) WHERE deleted_at IS NULL AND custom_display_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS IDX_order_region_id ON "order"(region_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_customer_id ON "order"(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_sales_channel_id ON "order"(sales_channel_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_currency_code ON "order"(currency_code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_shipping_address_id ON "order"(shipping_address_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_billing_address_id ON "order"(billing_address_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_is_draft_order ON "order"(is_draft_order) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS order_line_item (
 id TEXT PRIMARY KEY, totals_id TEXT, title TEXT NOT NULL, subtitle TEXT, thumbnail TEXT, variant_id TEXT, product_id TEXT,
 product_title TEXT, product_description TEXT, product_subtitle TEXT, product_type TEXT, product_type_id TEXT, product_collection TEXT,
 product_handle TEXT, variant_sku TEXT, variant_barcode TEXT, variant_title TEXT, variant_option_values JSONB,
 requires_shipping BOOLEAN NOT NULL DEFAULT true, is_giftcard BOOLEAN NOT NULL DEFAULT false,
 is_discountable BOOLEAN NOT NULL DEFAULT true, is_tax_inclusive BOOLEAN NOT NULL DEFAULT false,
 compare_at_unit_price NUMERIC, raw_compare_at_unit_price JSONB, unit_price NUMERIC NOT NULL, raw_unit_price JSONB NOT NULL,
 is_custom_price BOOLEAN NOT NULL DEFAULT false, metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_line_item_product_id ON order_line_item(product_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_line_item_variant_id ON order_line_item(variant_id) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS order_item (
 id TEXT PRIMARY KEY, order_id TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1, item_id TEXT NOT NULL REFERENCES order_line_item(id) ON UPDATE CASCADE ON DELETE CASCADE,
 unit_price NUMERIC, raw_unit_price JSONB, compare_at_unit_price NUMERIC, raw_compare_at_unit_price JSONB,
 quantity NUMERIC NOT NULL, raw_quantity JSONB NOT NULL, fulfilled_quantity NUMERIC NOT NULL DEFAULT 0, raw_fulfilled_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
 delivered_quantity NUMERIC NOT NULL DEFAULT 0, raw_delivered_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
 shipped_quantity NUMERIC NOT NULL DEFAULT 0, raw_shipped_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
 return_requested_quantity NUMERIC NOT NULL DEFAULT 0, raw_return_requested_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
 return_received_quantity NUMERIC NOT NULL DEFAULT 0, raw_return_received_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
 return_dismissed_quantity NUMERIC NOT NULL DEFAULT 0, raw_return_dismissed_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
 written_off_quantity NUMERIC NOT NULL DEFAULT 0, raw_written_off_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}', metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_item_order_id_version ON order_item(order_id,version) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS order_summary (
 id TEXT PRIMARY KEY, order_id TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1, totals JSONB NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_summary_order_id_version ON order_summary(order_id,version) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS order_change (
 id TEXT PRIMARY KEY, order_id TEXT NOT NULL, return_id TEXT, claim_id TEXT, exchange_id TEXT, version INTEGER NOT NULL,
 change_type TEXT, description TEXT, status TEXT DEFAULT 'pending', internal_note TEXT, created_by TEXT, requested_by TEXT,
 requested_at TIMESTAMPTZ, confirmed_by TEXT, confirmed_at TIMESTAMPTZ, declined_by TEXT, declined_reason TEXT,
 metadata JSONB, declined_at TIMESTAMPTZ, canceled_by TEXT, canceled_at TIMESTAMPTZ, carry_over_promotions BOOLEAN,
 no_notification BOOLEAN, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_change_order_id_version ON order_change(order_id,version) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS order_change_action (
 id TEXT PRIMARY KEY, order_id TEXT NOT NULL, return_id TEXT, claim_id TEXT, exchange_id TEXT, ordering BIGSERIAL NOT NULL,
 version INTEGER, order_change_id TEXT REFERENCES order_change(id) ON UPDATE CASCADE ON DELETE CASCADE,
 reference TEXT, reference_id TEXT, action TEXT NOT NULL, details JSONB NOT NULL DEFAULT '{}', amount NUMERIC, raw_amount JSONB,
 internal_note TEXT, applied BOOLEAN NOT NULL DEFAULT false, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_change_action_order_change_id ON order_change_action(order_change_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_change_action_order_id ON order_change_action(order_id) WHERE deleted_at IS NULL;
