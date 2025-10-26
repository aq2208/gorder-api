package cache

import (
	"context"
	"time"

	"github.com/aq2208/gorder-api/internal/usecase"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisCache(rdb *redis.Client, ttl time.Duration) *RedisCache {
	return &RedisCache{rdb: rdb, ttl: ttl}
}

func (r RedisCache) SetStatus(ctx context.Context, orderID string, status string) error {
	key := "order:status:" + orderID
	if r.ttl > 0 {
		return r.rdb.Set(ctx, key, status, time.Duration(r.ttl)*time.Second).Err()
	}
	return r.rdb.Set(ctx, key, status, 0).Err()
}

func (r RedisCache) GetStatus(ctx context.Context, orderID string) (string, error) {
	//TODO implement me
	panic("implement me")
}

var _ usecase.OrderCache = (*RedisCache)(nil)
