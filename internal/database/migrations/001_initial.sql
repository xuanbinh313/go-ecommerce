CREATE TABLE IF NOT EXISTS auth_identities (
    id TEXT PRIMARY KEY,
    app_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS provider_identities (
    id TEXT PRIMARY KEY,
    auth_identity_id TEXT NOT NULL REFERENCES auth_identities(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    user_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    provider_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, entity_id)
);

CREATE TABLE IF NOT EXISTS customers (
    id TEXT PRIMARY KEY,
    company_name TEXT,
    first_name TEXT,
    last_name TEXT,
    email TEXT,
    phone TEXT,
    has_account BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS customer_addresses (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    address_name TEXT,
    is_default_shipping BOOLEAN NOT NULL DEFAULT false,
    is_default_billing BOOLEAN NOT NULL DEFAULT false,
    company TEXT,
    first_name TEXT,
    last_name TEXT,
    address_1 TEXT,
    address_2 TEXT,
    city TEXT,
    country_code TEXT,
    province TEXT,
    postal_code TEXT,
    phone TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

