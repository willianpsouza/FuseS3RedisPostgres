CREATE TABLE IF NOT EXISTS objects (
  id BIGSERIAL NOT NULL,
  date_partition DATE NOT NULL,
  virtual_path TEXT NOT NULL,
  path_hash CHAR(64) NOT NULL,
  filename TEXT NOT NULL,
  filename_hash CHAR(64) NOT NULL,
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
  status TEXT NOT NULL DEFAULT 'active',
  PRIMARY KEY (id, date_partition),
  UNIQUE (date_partition, path_hash, filename_hash)
) PARTITION BY RANGE (date_partition);

CREATE TABLE IF NOT EXISTS objects_default PARTITION OF objects DEFAULT;

CREATE INDEX IF NOT EXISTS idx_objects_date_partition ON objects(date_partition);
CREATE INDEX IF NOT EXISTS idx_objects_path_hash_filename_hash ON objects(path_hash, filename_hash);
