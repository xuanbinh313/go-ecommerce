CREATE TABLE IF NOT EXISTS property_label (
    id TEXT PRIMARY KEY,
    entity TEXT NOT NULL,
    property TEXT NOT NULL,
    label TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_property_label_entity_property_unique ON property_label(entity, property) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_property_label_entity ON property_label(entity) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_property_label_deleted_at ON property_label(deleted_at) WHERE deleted_at IS NULL;
