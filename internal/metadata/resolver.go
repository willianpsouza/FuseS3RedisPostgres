package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/example/fuses3redispostgres/internal/cache"
	"github.com/redis/go-redis/v9"
)

type Resolver struct {
	repo  *Repository
	redis *redis.Client
	lru   *cache.LRU[string, Object]
	ttl   time.Duration
}

func NewResolver(repo *Repository, redis *redis.Client, cap int, ttl time.Duration) *Resolver {
	return &Resolver{repo: repo, redis: redis, lru: cache.NewLRU[string, Object](cap), ttl: ttl}
}

func (r *Resolver) Resolve(ctx context.Context, virtualPath string) (Object, error) {
	vp := normalizeVirtualPath(virtualPath)
	if obj, ok := r.lru.Get(vp); ok {
		return obj, nil
	}
	key := "resolve:path:" + vp
	if raw, err := r.redis.Get(ctx, key).Result(); err == nil {
		var obj Object
		if uerr := json.Unmarshal([]byte(raw), &obj); uerr == nil {
			r.lru.Set(vp, obj)
			return obj, nil
		}
	}
	obj, err := r.repo.ResolveByPath(ctx, vp)
	if err != nil {
		return Object{}, err
	}
	r.lru.Set(vp, obj)
	b, _ := json.Marshal(obj)
	if err := r.redis.Set(ctx, key, b, r.ttl).Err(); err != nil {
		return obj, fmt.Errorf("set redis cache: %w", err)
	}
	return obj, nil
}

func JoinVirtualPath(baseDir, name string) string {
	return normalizeVirtualPath(path.Join(baseDir, name))
}
