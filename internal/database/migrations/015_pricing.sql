CREATE TABLE IF NOT EXISTS price_set (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS price_list (
    id TEXT PRIMARY KEY, title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'draft', type TEXT NOT NULL DEFAULT 'sale',
    starts_at TIMESTAMPTZ, ends_at TIMESTAMPTZ, rules_count INTEGER DEFAULT 0, metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_price_list_active ON price_list(id,status,starts_at,ends_at) WHERE deleted_at IS NULL AND status='active';
CREATE TABLE IF NOT EXISTS price (
    id TEXT PRIMARY KEY, title TEXT, currency_code TEXT NOT NULL, amount NUMERIC NOT NULL, raw_amount JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
    min_quantity NUMERIC, raw_min_quantity JSONB, max_quantity NUMERIC, raw_max_quantity JSONB, rules_count INTEGER DEFAULT 0,
    price_set_id TEXT NOT NULL REFERENCES price_set(id) ON UPDATE CASCADE ON DELETE CASCADE,
    price_list_id TEXT REFERENCES price_list(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_price_set_id ON price(price_set_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_price_list_id ON price(price_list_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_price_currency_code ON price(currency_code) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS price_rule (
    id TEXT PRIMARY KEY, attribute TEXT NOT NULL, value TEXT NOT NULL, operator TEXT NOT NULL DEFAULT 'eq', priority INTEGER NOT NULL DEFAULT 0,
    price_id TEXT NOT NULL REFERENCES price(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_price_rule_unique ON price_rule(price_id,attribute,operator) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_price_rule_attribute ON price_rule(attribute) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_price_rule_attribute_value ON price_rule(attribute,value) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_price_rule_operator_value ON price_rule(operator,value) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS price_list_rule (
    id TEXT PRIMARY KEY, attribute TEXT NOT NULL, value JSONB,
    price_list_id TEXT NOT NULL REFERENCES price_list(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_price_list_rule_list ON price_list_rule(price_list_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_price_list_rule_attribute ON price_list_rule(attribute) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_price_list_rule_value ON price_list_rule USING GIN(value) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS price_preference (
    id TEXT PRIMARY KEY, attribute TEXT NOT NULL, value TEXT, is_tax_inclusive BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_price_preference_attribute_value ON price_preference(attribute,value) WHERE deleted_at IS NULL;
