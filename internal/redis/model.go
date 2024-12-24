package redis

import (
	"context"
	"time"
)

type MemStorage interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
}
