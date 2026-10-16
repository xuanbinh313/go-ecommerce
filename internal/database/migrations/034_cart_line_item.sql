CREATE TABLE IF NOT EXISTS cart_line_item (
    id TEXT PRIMARY KEY,
    cart_id TEXT NOT NULL,
    title TEXT NOT NULL,
    subtitle TEXT,
    thumbnail TEXT,
    quantity INTEGER NOT NULL,
    variant_id TEXT,
    product_id TEXT,
    product_title TEXT,
    product_description TEXT,
    product_subtitle TEXT,
    product_type TEXT,
    product_type_id TEXT,
    product_collection TEXT,
    product_handle TEXT,
    variant_sku TEXT,
    variant_barcode TEXT,
    variant_title TEXT,
    variant_option_values JSONB,
    requires_shipping BOOLEAN NOT NULL DEFAULT TRUE,
    is_discountable BOOLEAN NOT NULL DEFAULT TRUE,
    is_giftcard BOOLEAN NOT NULL DEFAULT FALSE,
    is_tax_inclusive BOOLEAN NOT NULL DEFAULT FALSE,
    is_custom_price BOOLEAN NOT NULL DEFAULT FALSE,
    unit_price NUMERIC NOT NULL,
    raw_unit_price JSONB NOT NULL,
    compare_at_unit_price NUMERIC,
    raw_compare_at_unit_price JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT cart_line_item_unit_price_check CHECK (unit_price >= 0)
);

CREATE INDEX IF NOT EXISTS IDX_line_item_cart_id ON cart_line_item(cart_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_line_item_product_id ON cart_line_item(product_id) WHERE deleted_at IS NULL AND product_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS IDX_line_item_variant_id ON cart_line_item(variant_id) WHERE deleted_at IS NULL AND variant_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS IDX_line_item_product_type_id ON cart_line_item(product_type_id) WHERE deleted_at IS NULL AND product_type_id IS NOT NULL;
