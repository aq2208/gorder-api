package cache

import (
	"context"
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

func (s *RedisIdempotencyStore) TryLock(ctx context.Context, scope, key string) (bool, error) {
	return s.rdb.SetNX(ctx, "idemp:"+scope+":"+key, "1", s.ttl).Result()
}

func (s *RedisIdempotencyStore) Remember(ctx context.Context, scope, key, value string) error {
	return s.rdb.Set(ctx, "idemp:map:"+scope+":"+key, value, s.ttl).Err()
}

func (s *RedisIdempotencyStore) Recall(ctx context.Context, scope, key string) (string, bool, error) {
	val, err := s.rdb.Get(ctx, "idemp:map:"+scope+":"+key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	return val, true, err
}

var _ usecase.IdempotencyStore = (*RedisIdempotencyStore)(nil)
