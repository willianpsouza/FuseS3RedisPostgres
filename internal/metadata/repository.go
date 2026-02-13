package metadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Object struct {
	Filename     string    `json:"filename"`
	Bucket       string    `json:"bucket"`
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	ETag         string    `json:"etag"`
	LastModified time.Time `json:"last_modified"`
	StorageClass string    `json:"storage_class"`
	VersionID    *string   `json:"version_id,omitempty"`
	ChecksumMD5  *string   `json:"checksum_md5,omitempty"`
	ChecksumSHA  *string   `json:"checksum_sha256,omitempty"`
}

var ErrNotFound = errors.New("object not found")

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func FilenameHash(name string) string {
	s := sha256.Sum256([]byte(name))
	return hex.EncodeToString(s[:])
}

func (r *Repository) ResolveByFilename(ctx context.Context, filename string) (Object, error) {
	q := `SELECT filename,bucket,key,size,etag,last_modified,storage_class,version_id,checksum_md5,checksum_sha256
	FROM objects WHERE filename_hash=$1 LIMIT 1`
	obj := Object{}
	err := r.pool.QueryRow(ctx, q, FilenameHash(filename)).Scan(
		&obj.Filename, &obj.Bucket, &obj.Key, &obj.Size, &obj.ETag, &obj.LastModified,
		&obj.StorageClass, &obj.VersionID, &obj.ChecksumMD5, &obj.ChecksumSHA,
	)
	if err != nil {
		if errors.Is(err, pgxpool.ErrClosed) {
			return Object{}, fmt.Errorf("pool closed: %w", err)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return Object{}, ErrNotFound
		}
		return Object{}, fmt.Errorf("query resolve: %w", err)
	}
	return obj, nil
}

func (r *Repository) UpsertObject(ctx context.Context, obj Object, datePartition time.Time, status string) error {
	q := `INSERT INTO objects
	(filename,filename_hash,date_partition,bucket,key,size,etag,last_modified,checksum_md5,checksum_sha256,status)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	ON CONFLICT (filename_hash)
	DO UPDATE SET bucket=EXCLUDED.bucket,key=EXCLUDED.key,size=EXCLUDED.size,etag=EXCLUDED.etag,
	last_modified=EXCLUDED.last_modified,checksum_md5=EXCLUDED.checksum_md5,checksum_sha256=EXCLUDED.checksum_sha256,status=EXCLUDED.status,verified_at=NOW()`
	_, err := r.pool.Exec(ctx, q, obj.Filename, FilenameHash(obj.Filename), datePartition, obj.Bucket, obj.Key, obj.Size, obj.ETag, obj.LastModified, obj.ChecksumMD5, obj.ChecksumSHA, status)
	if err != nil {
		return fmt.Errorf("upsert object: %w", err)
	}
	return nil
}
