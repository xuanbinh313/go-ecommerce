CREATE TABLE IF NOT EXISTS shipping_option_type (
 id TEXT PRIMARY KEY,
 label TEXT NOT NULL,
 description TEXT,
 code TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_shipping_option_type_code ON shipping_option_type(code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_shipping_option_type_label ON shipping_option_type(label) WHERE deleted_at IS NULL;
