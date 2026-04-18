package cache

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr string) *RedisStore {
	return &RedisStore{
		client: redis.NewClient(&redis.Options{
			Addr: strings.TrimSpace(addr),
		}),
	}
}

func (s *RedisStore) PreferredBackend(ctx context.Context, key string) (string, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false, nil
	}

	value, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(value) == "" {
		return "", false, nil
	}
	return value, true, nil
}

func (s *RedisStore) RememberBackend(ctx context.Context, key, backend string, ttl time.Duration) error {
	key = strings.TrimSpace(key)
	backend = strings.TrimSpace(backend)
	if key == "" || backend == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return s.client.Set(ctx, key, backend, ttl).Err()
}
