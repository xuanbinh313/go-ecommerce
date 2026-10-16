CREATE TABLE IF NOT EXISTS promotion_campaign (
    id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT, campaign_identifier TEXT NOT NULL, starts_at TIMESTAMPTZ, ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_promotion_campaign_identifier ON promotion_campaign(campaign_identifier) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS promotion_campaign_budget (
    id TEXT PRIMARY KEY, campaign_id TEXT NOT NULL REFERENCES promotion_campaign(id) ON UPDATE CASCADE ON DELETE CASCADE,
    type TEXT NOT NULL, currency_code TEXT, "limit" NUMERIC, raw_limit JSONB, used NUMERIC NOT NULL DEFAULT 0, raw_used JSONB NOT NULL DEFAULT '{"value":"0","precision":20}', attribute TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS promotion_campaign_budget_usage (
    id TEXT PRIMARY KEY, budget_id TEXT NOT NULL REFERENCES promotion_campaign_budget(id) ON UPDATE CASCADE ON DELETE CASCADE,
    attribute_value TEXT NOT NULL, used NUMERIC NOT NULL DEFAULT 0, raw_used JSONB NOT NULL DEFAULT '{"value":"0","precision":20}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_campaign_budget_usage_attribute_budget ON promotion_campaign_budget_usage(attribute_value,budget_id) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS promotion (
    id TEXT PRIMARY KEY, code TEXT NOT NULL, is_automatic BOOLEAN NOT NULL DEFAULT false, is_tax_inclusive BOOLEAN NOT NULL DEFAULT false,
    "limit" INTEGER, used INTEGER NOT NULL DEFAULT 0, type TEXT NOT NULL CHECK(type IN ('standard','buyget')),
    status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','active','inactive')),
    campaign_id TEXT REFERENCES promotion_campaign(id) ON UPDATE CASCADE ON DELETE SET NULL, metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_unique_promotion_code ON promotion(code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_promotion_is_automatic ON promotion(is_automatic) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS promotion_application_method (
    id TEXT PRIMARY KEY, promotion_id TEXT NOT NULL REFERENCES promotion(id) ON UPDATE CASCADE ON DELETE CASCADE,
    value NUMERIC, raw_value JSONB, currency_code TEXT, max_quantity INTEGER, apply_to_quantity INTEGER, buy_rules_min_quantity INTEGER,
    type TEXT NOT NULL CHECK(type IN ('fixed','percentage')), target_type TEXT NOT NULL CHECK(target_type IN ('order','shipping_methods','items')),
    allocation TEXT CHECK(allocation IN ('each','across','once')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS promotion_rule (
    id TEXT PRIMARY KEY, description TEXT, attribute TEXT NOT NULL, operator TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_promotion_rule_attribute_operator ON promotion_rule(attribute,operator) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_promotion_rule_operator_attribute_id ON promotion_rule(operator,attribute,id) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS promotion_rule_value (
    id TEXT PRIMARY KEY, promotion_rule_id TEXT NOT NULL REFERENCES promotion_rule(id) ON UPDATE CASCADE ON DELETE CASCADE, value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_promotion_rule_value_rule_value ON promotion_rule_value(promotion_rule_id,value) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_promotion_rule_value_value ON promotion_rule_value(value) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS promotion_promotion_rule (
    id TEXT PRIMARY KEY, promotion_id TEXT NOT NULL REFERENCES promotion(id) ON UPDATE CASCADE ON DELETE CASCADE,
    rule_id TEXT NOT NULL REFERENCES promotion_rule(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_promotion_promotion_rule_pair ON promotion_promotion_rule(promotion_id,rule_id) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS application_method_target_rules (
    id TEXT PRIMARY KEY, application_method_id TEXT NOT NULL REFERENCES promotion_application_method(id) ON UPDATE CASCADE ON DELETE CASCADE,
    rule_id TEXT NOT NULL REFERENCES promotion_rule(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS application_method_buy_rules (
    id TEXT PRIMARY KEY, application_method_id TEXT NOT NULL REFERENCES promotion_application_method(id) ON UPDATE CASCADE ON DELETE CASCADE,
    rule_id TEXT NOT NULL REFERENCES promotion_rule(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
