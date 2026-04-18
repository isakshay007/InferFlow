package cache

import (
	"context"
	"time"
)

type Store interface {
	PreferredBackend(ctx context.Context, key string) (string, bool, error)
	RememberBackend(ctx context.Context, key, backend string, ttl time.Duration) error
}
