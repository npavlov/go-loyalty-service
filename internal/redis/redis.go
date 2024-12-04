package redis

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/npavlov/go-loyalty-service/internal/config"
)

type RStorage struct {
	client *redis.Client
}

func NewRStorage(cfg config.Config) *RStorage {
	//nolint:exhaustruct
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis, // use default Addr
		Password: "",        // no password set
		DB:       0,         // use default DB
	})

	return &RStorage{client: redisClient}
}

func (rst *RStorage) Ping(ctx context.Context) error {
	err := rst.client.Ping(ctx).Err()

	return errors.Wrap(err, "redis ping")
}

func (rst *RStorage) Get(ctx context.Context, key string) (string, error) {
	result, err := rst.client.Get(ctx, key).Result()
	if err != nil {
		return "", errors.Wrap(err, "failed to get value")
	}

	return result, nil
}

func (rst *RStorage) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	err := rst.client.Set(ctx, key, value, expiration).Err()

	return errors.Wrap(err, "failed to set value")
}
