CREATE TABLE IF NOT EXISTS stock_location_address (
    id TEXT PRIMARY KEY,
    address_1 TEXT NOT NULL,
    address_2 TEXT,
    company TEXT,
    city TEXT,
    country_code TEXT NOT NULL,
    phone TEXT,
    province TEXT,
    postal_code TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS stock_location (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    address_id TEXT REFERENCES stock_location_address(id) ON UPDATE CASCADE ON DELETE CASCADE,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_stock_location_address_id ON stock_location(address_id) WHERE deleted_at IS NULL;
