package redis

import (
	"context"
	"time"
)

type MemStorage interface {
	Ping(ctx context.Context) error
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
}
