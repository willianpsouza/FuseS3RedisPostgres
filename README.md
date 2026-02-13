# FuseS3RedisPostgres

Production-ready Go monorepo skeleton for a virtual filesystem on top of S3 with Postgres index and Redis cache/coordination.

## Components
- `cmd/fusefs`: read-only FUSE mount (`/files/<filename>` and `/by-date/...`) using `go-fuse/v2`.
- `cmd/ingest-api`: ingestion API using **Gin** (fast, mature middleware ecosystem, easy streaming/multipart handling).
- `cmd/scanner-agent`: legacy-side agent that scans local dirs and streams files to ingest API.

## Architecture highlights
- No S3 LIST calls in hot path.
- Fast resolution via composite path+filename hashes (`path_hash`,`filename_hash`) and date partitions in Postgres.
- Two-level metadata cache in resolver: local LRU + Redis.
- S3 read path via Range GET with block/prefetch settings and concurrency limits (global and per-bucket).
- Structured logging (`zap`), health checks, Prometheus metrics endpoint.

## Local run
1. Copy env:
   ```bash
   cp .env.example .env
   ```
2. Start infra + API:
   ```bash
   make up
   ```
3. Run migration:
   ```bash
   make migrate
   ```
4. Build binaries:
   ```bash
   make build
   ```

## FUSE usage example
```bash
sudo mkdir -p /mnt/virtualfs
export FUSE_MOUNT_POINT=/mnt/virtualfs
./fusefs
ls /mnt/virtualfs/files
```

## API examples
Multipart:
```bash
curl -X POST "http://localhost:8080/v1/upload?date=2024-01-01&path=/20200101/2014/file.txt" \
  -H "X-API-Key: changeme" \
  -F "file=@./file.txt"
```

Streaming:
```bash
curl -X POST "http://localhost:8080/v1/upload?date=2024-01-01&path=/20200101/2014/file.txt" \
  -H "X-API-Key: changeme" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @./file.txt
```

Resolve:
```bash
curl "http://localhost:8080/v1/resolve?path=/20200101/2014/file.txt" -H "X-API-Key: changeme"
```

## Tuning
- `BLOCK_SIZE_BYTES` (default 8 MiB)
- `PREFETCH_SIZE_BYTES` (default 32 MiB)
- `GLOBAL_S3_LIMIT` and `PER_BUCKET_S3_LIMIT`
- `CACHE_SIZE_BYTES` and `CACHE_DIR`

## Systemd
See `deploy/systemd/fusefs.service` and `deploy/systemd/scanner-agent.service`.

## Tests
```bash
make test
```

## Notes
- `readdir` is intentionally limited/safe and does not enumerate huge datasets.
- Prepared for Localstack-based integration tests (not mandatory by default).


## Partitioning strategy
- `objects` is partitioned by `date_partition` (RANGE), with default partition enabled.
- Uniqueness is enforced by `(date_partition, path_hash, filename_hash)`, allowing repeated filenames under different paths safely.
