CREATE TABLE IF NOT EXISTS api_key (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    token TEXT NOT NULL,
    salt TEXT NOT NULL DEFAULT '',
    redacted TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('publishable', 'secret')),
    last_used_at TIMESTAMPTZ,
    created_by TEXT NOT NULL,
    revoked_by TEXT,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_api_key_token_unique ON api_key(token);
CREATE INDEX IF NOT EXISTS IDX_api_key_type ON api_key(type);
CREATE INDEX IF NOT EXISTS IDX_api_key_redacted ON api_key(redacted);
CREATE INDEX IF NOT EXISTS IDX_api_key_revoked_at ON api_key(revoked_at);
