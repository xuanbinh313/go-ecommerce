DO $$ BEGIN
  IF to_regclass('auth_identities') IS NOT NULL AND to_regclass('auth_identity') IS NULL THEN
    ALTER TABLE auth_identities RENAME TO auth_identity;
  END IF;
END $$;
DO $$ BEGIN
  IF to_regclass('provider_identities') IS NOT NULL AND to_regclass('provider_identity') IS NULL THEN
    ALTER TABLE provider_identities RENAME TO provider_identity;
  END IF;
END $$;
DO $$ BEGIN
  IF to_regclass('customers') IS NOT NULL AND to_regclass('customer') IS NULL THEN
    ALTER TABLE customers RENAME TO customer;
  END IF;
END $$;
DO $$ BEGIN
  IF to_regclass('customer_addresses') IS NOT NULL AND to_regclass('customer_address') IS NULL THEN
    ALTER TABLE customer_addresses RENAME TO customer_address;
  END IF;
END $$;
ALTER TABLE provider_identity ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE provider_identity ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE customer_address ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
DROP INDEX IF EXISTS customer_one_default_billing;
DROP INDEX IF EXISTS customer_one_default_shipping;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_customer_address_unique_customer_billing ON customer_address(customer_id) WHERE is_default_billing AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_customer_address_unique_customer_shipping ON customer_address(customer_id) WHERE is_default_shipping AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS auth_verification (
  id TEXT PRIMARY KEY,
  auth_identity_id TEXT NOT NULL REFERENCES auth_identity(id) ON DELETE CASCADE,
  entity_id TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  code_provider TEXT NOT NULL,
  verified_at TIMESTAMPTZ,
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  provider_metadata JSONB,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_auth_verification_unique_auth_identity_entity_id_entity_type
  ON auth_verification(auth_identity_id,entity_id,entity_type) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS auth_password_reset_token (
  id TEXT PRIMARY KEY,
  auth_identity_id TEXT NOT NULL REFERENCES auth_identity(id) ON DELETE CASCADE,
  provider_identity_id TEXT REFERENCES provider_identity(id) ON DELETE CASCADE,
  entity_id TEXT NOT NULL,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_auth_password_reset_token_provider_identity_id ON auth_password_reset_token(provider_identity_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_auth_password_reset_token_token_hash ON auth_password_reset_token(token_hash) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_auth_password_reset_token_expires_at ON auth_password_reset_token(expires_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS auth_mfa_factor (
  id TEXT PRIMARY KEY,
  auth_identity_id TEXT NOT NULL REFERENCES auth_identity(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  status TEXT NOT NULL,
  provider_metadata JSONB,
  metadata JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_auth_mfa_factor_auth_identity_id ON auth_mfa_factor(auth_identity_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_auth_mfa_factor_auth_identity_provider_active ON auth_mfa_factor(auth_identity_id,provider) WHERE deleted_at IS NULL AND status IN ('pending','enabled');

CREATE TABLE IF NOT EXISTS auth_mfa_recovery_code (
  id TEXT PRIMARY KEY,
  auth_identity_id TEXT NOT NULL REFERENCES auth_identity(id) ON DELETE CASCADE,
  code_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_auth_mfa_recovery_code_auth_identity_id ON auth_mfa_recovery_code(auth_identity_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_auth_mfa_recovery_code_auth_identity_code_hash ON auth_mfa_recovery_code(auth_identity_id,code_hash) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS customer_group (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS customer_group_name_unique ON customer_group(name) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS customer_group_customer (
  id TEXT PRIMARY KEY,
  customer_id TEXT NOT NULL REFERENCES customer(id) ON DELETE CASCADE,
  customer_group_id TEXT NOT NULL REFERENCES customer_group(id) ON DELETE CASCADE,
  created_by TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);
