CREATE TABLE IF NOT EXISTS locale (
    id TEXT PRIMARY KEY, code TEXT NOT NULL, name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_locale_code ON locale(code) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS translation_settings (
    id TEXT PRIMARY KEY, entity_type TEXT NOT NULL, fields JSONB NOT NULL, is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_translation_settings_entity_type ON translation_settings(entity_type) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS translation (
    id TEXT PRIMARY KEY, reference_id TEXT NOT NULL, reference TEXT NOT NULL, locale_code TEXT NOT NULL,
    translations JSONB NOT NULL, translated_field_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(), deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_translation_reference_id_locale_code ON translation(reference_id,locale_code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_translation_reference_id_reference_locale ON translation(reference_id,reference,locale_code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_translation_reference_locale ON translation(reference,locale_code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_translation_reference_id_reference ON translation(reference_id,reference) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_translation_locale_code ON translation(locale_code) WHERE deleted_at IS NULL;
