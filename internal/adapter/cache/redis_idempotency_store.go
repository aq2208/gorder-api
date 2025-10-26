package cache

import (
	"context"
	"errors"
	"time"

	"github.com/aq2208/gorder-api/internal/usecase"
	"github.com/redis/go-redis/v9"
)

type RedisIdempotencyStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisIdempotencyStore(rdb *redis.Client, ttl time.Duration) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{rdb: rdb, ttl: ttl}
}

func (s *RedisIdempotencyStore) TryLock(ctx context.Context, key string) (bool, error) {
	return s.rdb.SetNX(ctx, "idemp:"+key, "1", s.ttl).Result()
}

func (s *RedisIdempotencyStore) Remember(ctx context.Context, key, value string) error {
	return s.rdb.Set(ctx, "idemp:map:"+key, value, s.ttl).Err()
}

func (s *RedisIdempotencyStore) Recall(ctx context.Context, key string) (string, bool, error) {
	val, err := s.rdb.Get(ctx, "idemp:map:"+key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	return val, true, err
}

var _ usecase.IdempotencyStore = (*RedisIdempotencyStore)(nil)
