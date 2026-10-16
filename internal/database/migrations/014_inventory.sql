CREATE TABLE IF NOT EXISTS inventory_item (
    id TEXT PRIMARY KEY,
    sku TEXT,
    origin_country TEXT,
    hs_code TEXT,
    mid_code TEXT,
    material TEXT,
    unit_of_measure TEXT,
    weight INTEGER,
    length INTEGER,
    height INTEGER,
    width INTEGER,
    requires_shipping BOOLEAN NOT NULL DEFAULT true,
    description TEXT,
    title TEXT,
    thumbnail TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_inventory_item_sku ON inventory_item(sku) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS inventory_level (
    id TEXT PRIMARY KEY,
    inventory_item_id TEXT NOT NULL REFERENCES inventory_item(id) ON UPDATE CASCADE ON DELETE CASCADE,
    location_id TEXT NOT NULL,
    stocked_quantity NUMERIC NOT NULL DEFAULT 0,
    reserved_quantity NUMERIC NOT NULL DEFAULT 0,
    incoming_quantity NUMERIC NOT NULL DEFAULT 0,
    raw_stocked_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
    raw_reserved_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
    raw_incoming_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_inventory_level_item ON inventory_level(inventory_item_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_inventory_level_location ON inventory_level(location_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_inventory_level_location_id_inventory_item_id ON inventory_level(inventory_item_id, location_id) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS reservation_item (
    id TEXT PRIMARY KEY,
    line_item_id TEXT,
    allow_backorder BOOLEAN NOT NULL DEFAULT false,
    location_id TEXT NOT NULL,
    quantity NUMERIC NOT NULL,
    raw_quantity JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
    external_id TEXT,
    description TEXT,
    created_by TEXT,
    metadata JSONB,
    inventory_item_id TEXT NOT NULL REFERENCES inventory_item(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_reservation_item_line_item ON reservation_item(line_item_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_reservation_item_location ON reservation_item(location_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_reservation_item_item ON reservation_item(inventory_item_id) WHERE deleted_at IS NULL;
