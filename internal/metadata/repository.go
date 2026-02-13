package metadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Object struct {
	VirtualPath  string    `json:"virtual_path"`
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

func hash(in string) string {
	s := sha256.Sum256([]byte(in))
	return hex.EncodeToString(s[:])
}

func normalizeVirtualPath(vpath string) string {
	if vpath == "" {
		return "/"
	}
	clean := path.Clean("/" + vpath)
	if clean == "." {
		return "/"
	}
	return clean
}

func (r *Repository) ResolveByPath(ctx context.Context, vpath string) (Object, error) {
	vp := normalizeVirtualPath(vpath)
	filename := path.Base(vp)
	q := `SELECT virtual_path,filename,bucket,key,size,etag,last_modified,storage_class,version_id,checksum_md5,checksum_sha256
	FROM objects WHERE path_hash=$1 AND filename_hash=$2 ORDER BY date_partition DESC LIMIT 1`
	obj := Object{}
	err := r.pool.QueryRow(ctx, q, hash(vp), hash(filename)).Scan(
		&obj.VirtualPath, &obj.Filename, &obj.Bucket, &obj.Key, &obj.Size, &obj.ETag, &obj.LastModified,
		&obj.StorageClass, &obj.VersionID, &obj.ChecksumMD5, &obj.ChecksumSHA,
	)
	if err != nil {
		if errors.Is(err, pgxpool.ErrClosed) {
			return Object{}, fmt.Errorf("pool closed: %w", err)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return Object{}, ErrNotFound
		}
		return Object{}, fmt.Errorf("query resolve by path: %w", err)
	}
	return obj, nil
}

func (r *Repository) UpsertObject(ctx context.Context, obj Object, datePartition time.Time, status string) error {
	obj.VirtualPath = normalizeVirtualPath(obj.VirtualPath)
	obj.Filename = path.Base(obj.VirtualPath)
	q := `INSERT INTO objects
	(date_partition,virtual_path,path_hash,filename,filename_hash,bucket,key,size,etag,last_modified,checksum_md5,checksum_sha256,status)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	ON CONFLICT (date_partition,path_hash,filename_hash)
	DO UPDATE SET bucket=EXCLUDED.bucket,key=EXCLUDED.key,size=EXCLUDED.size,etag=EXCLUDED.etag,
	last_modified=EXCLUDED.last_modified,checksum_md5=EXCLUDED.checksum_md5,checksum_sha256=EXCLUDED.checksum_sha256,status=EXCLUDED.status,verified_at=NOW()`
	_, err := r.pool.Exec(ctx, q, datePartition, obj.VirtualPath, hash(obj.VirtualPath), obj.Filename, hash(obj.Filename), obj.Bucket, obj.Key, obj.Size, obj.ETag, obj.LastModified, obj.ChecksumMD5, obj.ChecksumSHA, status)
	if err != nil {
		return fmt.Errorf("upsert object: %w", err)
	}
	return nil
}
