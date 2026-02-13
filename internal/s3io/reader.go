package s3io

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/sync/semaphore"
)

type Reader struct {
	client   *s3.Client
	global   *semaphore.Weighted
	perBkt   map[string]*semaphore.Weighted
	perLimit int64
}

func NewReader(client *s3.Client, globalLimit, perBucketLimit int64) *Reader {
	return &Reader{client: client, global: semaphore.NewWeighted(globalLimit), perBkt: map[string]*semaphore.Weighted{}, perLimit: perBucketLimit}
}

func (r *Reader) bucketSem(bucket string) *semaphore.Weighted {
	if s, ok := r.perBkt[bucket]; ok {
		return s
	}
	s := semaphore.NewWeighted(r.perLimit)
	r.perBkt[bucket] = s
	return s
}

func (r *Reader) GetRange(ctx context.Context, bucket, key string, start, end int64) ([]byte, error) {
	if err := r.global.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("acquire global: %w", err)
	}
	defer r.global.Release(1)
	bs := r.bucketSem(bucket)
	if err := bs.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("acquire bucket: %w", err)
	}
	defer bs.Release(1)
	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: &key, Range: ptr("bytes=" + strconv.FormatInt(start, 10) + "-" + strconv.FormatInt(end, 10))})
	if err != nil {
		return nil, fmt.Errorf("get object range: %w", err)
	}
	defer out.Body.Close()
	buf, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return buf, nil
}

func ptr(s string) *string { return &s }
