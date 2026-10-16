-- Medusa Payment module's current model shape. This file is authored only;
-- runtime startup does not apply it.
CREATE TABLE IF NOT EXISTS payment_collection (
    id TEXT PRIMARY KEY,
    currency_code TEXT NOT NULL,
    amount NUMERIC NOT NULL,
    raw_amount JSONB NOT NULL,
    authorized_amount NUMERIC,
    raw_authorized_amount JSONB,
    captured_amount NUMERIC,
    raw_captured_amount JSONB,
    refunded_amount NUMERIC,
    raw_refunded_amount JSONB,
    completed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'not_paid' CHECK (status IN ('not_paid','awaiting','authorized','partially_authorized','canceled','failed','partially_captured','completed')),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS payment_provider (
    id TEXT PRIMARY KEY,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS payment_collection_payment_providers (
    payment_collection_id TEXT NOT NULL REFERENCES payment_collection(id) ON UPDATE CASCADE ON DELETE CASCADE,
    payment_provider_id TEXT NOT NULL REFERENCES payment_provider(id) ON UPDATE CASCADE ON DELETE CASCADE,
    PRIMARY KEY (payment_collection_id, payment_provider_id)
);

CREATE TABLE IF NOT EXISTS payment_session (
    id TEXT PRIMARY KEY,
    currency_code TEXT NOT NULL,
    amount NUMERIC NOT NULL,
    raw_amount JSONB NOT NULL,
    provider_id TEXT NOT NULL,
    data JSONB NOT NULL DEFAULT '{}',
    context JSONB,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('authorized','captured','pending','requires_more','error','canceled','pending_authorization')),
    authorized_at TIMESTAMPTZ,
    payment_collection_id TEXT NOT NULL REFERENCES payment_collection(id) ON UPDATE CASCADE ON DELETE CASCADE,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS payment (
    id TEXT PRIMARY KEY,
    amount NUMERIC NOT NULL,
    raw_amount JSONB NOT NULL,
    currency_code TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    data JSONB,
    metadata JSONB,
    captured_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    payment_collection_id TEXT NOT NULL REFERENCES payment_collection(id) ON UPDATE CASCADE ON DELETE CASCADE,
    payment_session_id TEXT NOT NULL REFERENCES payment_session(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS capture (
    id TEXT PRIMARY KEY,
    amount NUMERIC NOT NULL,
    raw_amount JSONB NOT NULL,
    payment_id TEXT NOT NULL REFERENCES payment(id) ON UPDATE CASCADE ON DELETE CASCADE,
    created_by TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS refund_reason (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS refund (
    id TEXT PRIMARY KEY,
    amount NUMERIC NOT NULL,
    raw_amount JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','succeeded','failed')),
    idempotency_key TEXT,
    payment_id TEXT NOT NULL REFERENCES payment(id) ON UPDATE CASCADE ON DELETE CASCADE,
    refund_reason_id TEXT REFERENCES refund_reason(id) ON UPDATE CASCADE ON DELETE SET NULL,
    note TEXT,
    created_by TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS account_holder (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL,
    external_id TEXT NOT NULL,
    email TEXT,
    data JSONB NOT NULL DEFAULT '{}',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS IDX_payment_session_payment_collection_id ON payment_session(payment_collection_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_payment_provider_deleted_at ON payment_provider(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_payment_provider_id ON payment(provider_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_payment_payment_collection_id ON payment(payment_collection_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_payment_payment_session_id ON payment(payment_session_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_payment_payment_session_id_unique ON payment(payment_session_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_capture_payment_id ON capture(payment_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_refund_payment_id ON refund(payment_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_refund_refund_reason_id ON refund(refund_reason_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_refund_reason_deleted_at ON refund_reason(deleted_at) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_refund_payment_id_idempotency_key_unique ON refund(payment_id,idempotency_key) WHERE idempotency_key IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_account_holder_deleted_at ON account_holder(deleted_at) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_account_holder_provider_id_external_id_unique ON account_holder(provider_id,external_id) WHERE deleted_at IS NULL;
