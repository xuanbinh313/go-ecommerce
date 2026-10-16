CREATE TABLE IF NOT EXISTS return_reason (
    id TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    label TEXT NOT NULL,
    description TEXT,
    metadata JSONB,
    parent_return_reason_id TEXT REFERENCES return_reason(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS IDX_return_reason_deleted_at ON return_reason(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS IDX_return_reason_value ON return_reason(value) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_return_reason_parent_return_reason_id ON return_reason(parent_return_reason_id) WHERE deleted_at IS NULL;
