-- Medusa search-postgres native catalog and text-search extensions.
-- Authored only: application startup must never execute migrations implicitly.
DO $$
BEGIN
  BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_trgm;
  EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'search-postgres: could not enable pg_trgm (%).', SQLERRM;
  END;
  BEGIN
    CREATE EXTENSION IF NOT EXISTS unaccent;
  EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'search-postgres: could not enable unaccent (%).', SQLERRM;
  END;
  BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'unaccent')
       AND NOT EXISTS (
         SELECT 1 FROM pg_ts_config WHERE cfgname = 'medusa_search_english'
       ) THEN
      CREATE TEXT SEARCH CONFIGURATION medusa_search_english (COPY = english);
      ALTER TEXT SEARCH CONFIGURATION medusa_search_english
        ALTER MAPPING FOR hword, hword_part, word
        WITH unaccent, english_stem;
    END IF;
  EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'search-postgres: could not create medusa_search_english (%).', SQLERRM;
  END;
END
$$;

CREATE TABLE IF NOT EXISTS search_postgres_index (
  name text PRIMARY KEY,
  table_name text NOT NULL,
  schema_hash text NOT NULL,
  plan jsonb NOT NULL,
  document_count integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
