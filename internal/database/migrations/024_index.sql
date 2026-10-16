-- Medusa Index module schema. Entity and relation partitions are created from
-- the loaded graph schema by the storage adapter; startup does not run SQL.
CREATE TABLE IF NOT EXISTS index_data (
  id text NOT NULL,
  name text NOT NULL,
  data jsonb NOT NULL DEFAULT '{}',
  staled_at timestamptz NULL,
  document_tsv tsvector NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL,
  PRIMARY KEY (id, name)
) PARTITION BY LIST (name);

CREATE TABLE IF NOT EXISTS index_relation (
  id bigserial NOT NULL,
  pivot text NOT NULL,
  parent_name text NOT NULL,
  parent_id text NOT NULL,
  child_name text NOT NULL,
  child_id text NOT NULL,
  staled_at timestamptz NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL,
  PRIMARY KEY (id, pivot),
  CONSTRAINT idx_index_relation_unique UNIQUE (parent_id, child_id, child_name, parent_name, pivot)
) PARTITION BY LIST (pivot);

CREATE TABLE IF NOT EXISTS index_metadata (
  id text PRIMARY KEY,
  entity text NOT NULL,
  fields text NOT NULL,
  fields_hash text NOT NULL,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','done','error')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_index_metadata_entity ON index_metadata(entity) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS index_sync (
  id text PRIMARY KEY,
  entity text NOT NULL,
  last_key text NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_index_sync_entity ON index_sync(entity) WHERE deleted_at IS NULL;

CREATE OR REPLACE FUNCTION update_index_document_tsv() RETURNS trigger AS $$
BEGIN
  NEW.document_tsv := to_tsvector('simple', COALESCE((SELECT string_agg(value, ' ') FROM jsonb_each_text(NEW.data)), ''));
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_update_index_document_tsv ON index_data;
CREATE TRIGGER trg_update_index_document_tsv BEFORE INSERT OR UPDATE ON index_data
FOR EACH ROW EXECUTE FUNCTION update_index_document_tsv();

CREATE INDEX IF NOT EXISTS idx_index_data_document_tsv ON index_data USING gin(document_tsv);
