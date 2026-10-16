CREATE TABLE IF NOT EXISTS user_preference (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_user_preference_user_id_key_unique ON user_preference(user_id, key) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_user_preference_user_id ON user_preference(user_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS layout_configuration (
    id TEXT PRIMARY KEY,
    zone TEXT NOT NULL,
    user_id TEXT,
    is_system_default BOOLEAN NOT NULL DEFAULT false,
    configuration JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_layout_configuration_zone_user_id_unique ON layout_configuration(zone, user_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS IDX_layout_configuration_zone_unique ON layout_configuration(zone) WHERE is_system_default = true AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS notification_provider (
    id TEXT PRIMARY KEY,
    handle TEXT NOT NULL,
    name TEXT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    channels TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS notification (
    id TEXT PRIMARY KEY,
    "to" TEXT NOT NULL,
    "from" TEXT,
    channel TEXT NOT NULL,
    template TEXT,
    data JSONB,
    provider_data JSONB,
    trigger_type TEXT,
    resource_id TEXT,
    resource_type TEXT,
    receiver_id TEXT,
    original_notification_id TEXT,
    idempotency_key TEXT,
    external_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failure')),
    provider_id TEXT REFERENCES notification_provider(id) ON UPDATE CASCADE ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_notification_idempotency_key_unique ON notification(idempotency_key) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_notification_receiver_id ON notification(receiver_id);
CREATE INDEX IF NOT EXISTS IDX_notification_deleted_at ON notification(deleted_at) WHERE deleted_at IS NULL;
