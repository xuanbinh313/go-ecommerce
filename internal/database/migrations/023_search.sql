-- Medusa Search module bookkeeping schema. Authored only; startup never applies SQL.
CREATE TABLE IF NOT EXISTS search_index (
  id text PRIMARY KEY,
  name text NOT NULL,
  active_version integer NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_search_index_name ON search_index(name) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS search_index_version (
  id text PRIMARY KEY,
  version integer NOT NULL,
  provider text NOT NULL,
  physical_name text NOT NULL,
  definition_hash text NOT NULL,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','building','ready','error')),
  search_index_id text NOT NULL REFERENCES search_index(id) ON UPDATE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_search_index_version_index_version ON search_index_version(search_index_id, version) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_search_index_version_index_id ON search_index_version(search_index_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS search_index_sync (
  id text PRIMARY KEY,
  job_id text NULL,
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','processing','done','failed','canceled')),
  filters jsonb NULL,
  last_key text NULL,
  resumable boolean NOT NULL DEFAULT true,
  documents_synced integer NOT NULL DEFAULT 0,
  started_at timestamptz NULL,
  completed_at timestamptz NULL,
  error text NULL,
  search_index_version_id text NOT NULL REFERENCES search_index_version(id) ON UPDATE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz NULL
);
CREATE INDEX IF NOT EXISTS idx_search_index_sync_version_id ON search_index_sync(search_index_version_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_search_index_sync_job_id ON search_index_sync(job_id) WHERE deleted_at IS NULL;
