CREATE TABLE IF NOT EXISTS tax_provider (
    id TEXT PRIMARY KEY, is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS tax_region (
    id TEXT PRIMARY KEY, country_code TEXT NOT NULL, province_code TEXT, provider_id TEXT REFERENCES tax_provider(id) ON UPDATE CASCADE,
    parent_id TEXT REFERENCES tax_region(id) ON UPDATE CASCADE ON DELETE CASCADE, metadata JSONB, created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ,
    CONSTRAINT CK_tax_region_provider_top_level CHECK(parent_id IS NULL OR provider_id IS NULL),
    CONSTRAINT CK_tax_region_country_top_level CHECK(parent_id IS NULL OR province_code IS NOT NULL)
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_tax_region_unique_country_province ON tax_region(country_code,province_code) WHERE deleted_at IS NULL AND province_code IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_tax_region_unique_country_nullable_province ON tax_region(country_code) WHERE deleted_at IS NULL AND province_code IS NULL;
CREATE TABLE IF NOT EXISTS tax_rate (
    id TEXT PRIMARY KEY, rate DOUBLE PRECISION, code TEXT NOT NULL, name TEXT NOT NULL, is_default BOOLEAN NOT NULL DEFAULT false,
    is_combinable BOOLEAN NOT NULL DEFAULT false, tax_region_id TEXT NOT NULL REFERENCES tax_region(id) ON UPDATE CASCADE ON DELETE CASCADE,
    metadata JSONB, created_by TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_tax_rate_tax_region_id ON tax_rate(tax_region_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_single_default_region ON tax_rate(tax_region_id) WHERE is_default=true AND deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS tax_rate_rule (
    id TEXT PRIMARY KEY, tax_rate_id TEXT NOT NULL REFERENCES tax_rate(id) ON UPDATE CASCADE ON DELETE CASCADE,
    reference TEXT NOT NULL, reference_id TEXT NOT NULL, metadata JSONB, created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_tax_rate_rule_reference_id ON tax_rate_rule(reference_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_tax_rate_rule_unique_rate_reference ON tax_rate_rule(tax_rate_id,reference_id) WHERE deleted_at IS NULL;
