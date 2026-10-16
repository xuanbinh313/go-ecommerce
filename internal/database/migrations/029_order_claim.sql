CREATE TABLE IF NOT EXISTS order_claim (
    id TEXT PRIMARY KEY,
    order_version INTEGER NOT NULL,
    display_id BIGSERIAL NOT NULL,
    type TEXT NOT NULL,
    no_notification BOOLEAN,
    refund_amount NUMERIC,
    raw_refund_amount JSONB,
    created_by TEXT,
    canceled_at TIMESTAMPTZ,
    metadata JSONB,
    order_id TEXT NOT NULL REFERENCES "order"(id) ON UPDATE CASCADE ON DELETE CASCADE,
    return_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_claim_display_id ON order_claim(display_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_claim_order_id ON order_claim(order_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_claim_return_id ON order_claim(return_id) WHERE return_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_claim_deleted_at ON order_claim(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS order_claim_item (
    id TEXT PRIMARY KEY,
    claim_id TEXT NOT NULL,
    item_id TEXT NOT NULL,
    is_additional_item BOOLEAN NOT NULL DEFAULT FALSE,
    reason TEXT CHECK(reason IS NULL OR reason IN ('missing_item','wrong_item','production_failure','other')),
    quantity NUMERIC NOT NULL,
    raw_quantity JSONB NOT NULL,
    note TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_claim_item_claim_id ON order_claim_item(claim_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_claim_item_item_id ON order_claim_item(item_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_claim_item_deleted_at ON order_claim_item(deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS order_claim_item_image (
    id TEXT PRIMARY KEY,
    claim_item_id TEXT NOT NULL,
    url TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_order_claim_item_image_claim_item_id ON order_claim_item_image(claim_item_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_order_claim_item_image_deleted_at ON order_claim_item_image(deleted_at) WHERE deleted_at IS NOT NULL;
