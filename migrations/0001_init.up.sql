CREATE TABLE IF NOT EXISTS objects (
  id BIGSERIAL PRIMARY KEY,
  filename TEXT NOT NULL,
  filename_hash CHAR(64) NOT NULL UNIQUE,
  date_partition DATE NOT NULL,
  bucket TEXT NOT NULL,
  key TEXT NOT NULL,
  size BIGINT NOT NULL DEFAULT 0,
  etag TEXT NOT NULL,
  last_modified TIMESTAMPTZ NOT NULL,
  storage_class TEXT,
  version_id TEXT,
  checksum_md5 TEXT,
  checksum_sha256 TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  verified_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'active'
);
CREATE INDEX IF NOT EXISTS idx_objects_date_partition ON objects(date_partition);
