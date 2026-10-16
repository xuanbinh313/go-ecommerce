CREATE TABLE IF NOT EXISTS shipping_profile (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_shipping_profile_name_unique ON shipping_profile(name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_shipping_profile_deleted_at ON shipping_profile(deleted_at) WHERE deleted_at IS NULL;
