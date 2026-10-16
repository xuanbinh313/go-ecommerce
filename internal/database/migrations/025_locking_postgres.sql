CREATE TABLE IF NOT EXISTS locking (
    id TEXT PRIMARY KEY,
    owner_id TEXT,
    expiration TIMESTAMPTZ
);
