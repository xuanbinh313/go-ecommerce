CREATE TABLE IF NOT EXISTS cart_address (
    id TEXT PRIMARY KEY, customer_id TEXT, company TEXT, first_name TEXT, last_name TEXT,
    address_1 TEXT, address_2 TEXT, city TEXT, country_code TEXT, province TEXT,
    postal_code TEXT, phone TEXT, metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS cart (
    id TEXT PRIMARY KEY, region_id TEXT, customer_id TEXT, sales_channel_id TEXT, email TEXT,
    currency_code TEXT NOT NULL, locale TEXT, shipping_address_id TEXT REFERENCES cart_address(id) ON UPDATE CASCADE ON DELETE SET NULL,
    billing_address_id TEXT REFERENCES cart_address(id) ON UPDATE CASCADE ON DELETE SET NULL,
    metadata JSONB, completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_cart_customer_id ON cart(customer_id) WHERE deleted_at IS NULL AND customer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS IDX_cart_region_id ON cart(region_id) WHERE deleted_at IS NULL AND region_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS IDX_cart_sales_channel_id ON cart(sales_channel_id) WHERE deleted_at IS NULL AND sales_channel_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS IDX_cart_currency_code ON cart(currency_code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_cart_shipping_address_id ON cart(shipping_address_id) WHERE deleted_at IS NULL AND shipping_address_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS IDX_cart_billing_address_id ON cart(billing_address_id) WHERE deleted_at IS NULL AND billing_address_id IS NOT NULL;
