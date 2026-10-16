ALTER TABLE provider_identity ADD COLUMN IF NOT EXISTS provider_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
DO $$ BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = ANY(current_schemas(false))
      AND table_name = 'provider_identity'
      AND column_name = 'password_hash'
  ) THEN
    EXECUTE 'UPDATE provider_identity SET provider_metadata=jsonb_set(provider_metadata,''{password}'',to_jsonb(password_hash),true) WHERE password_hash IS NOT NULL AND NOT (provider_metadata ? ''password'')';
  END IF;
END $$;
ALTER TABLE provider_identity DROP COLUMN IF EXISTS password_hash;
