package redis

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/npavlov/go-loyalty-service/internal/config"
	"github.com/npavlov/go-loyalty-service/internal/logger"
)

type RStorage struct {
	client *redis.Client
	tracer trace.Tracer
	logger *zerolog.Logger
}

func NewRStorage(cfg config.Config, logger *zerolog.Logger) *RStorage {
	// Initialize a Redis client
	//nolint:exhaustruct
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis, // use default Addr
		Password: "",        // no password set
		DB:       0,         // use default DB
	})

	// Initialize OpenTelemetry tracer
	tracer := otel.Tracer("redis")

	return &RStorage{
		client: redisClient,
		tracer: tracer,
		logger: logger,
	}
}

func (rst *RStorage) Ping(ctx context.Context) error {
	// Start a span for the Ping operation
	ctx, span := rst.tracer.Start(ctx, "Redis.Ping")
	defer span.End()

	log := logger.GetWithTrace(ctx, rst.logger)

	// Log the start of the operation
	log.Info().Msg("Pinging Redis server")

	err := rst.client.Ping(ctx).Err()
	// Record the result in the span
	if err != nil {
		span.RecordError(err)
		log.Error().Err(err).Msg("Redis ping failed")

		return errors.Wrap(err, "redis ping")
	}

	log.Info().Msg("Redis ping successful")

	return nil
}

func (rst *RStorage) Get(ctx context.Context, key string) (string, error) {
	// Start a span for the Get operation
	ctx, span := rst.tracer.Start(ctx, "Redis.Get")
	defer span.End()

	log := logger.GetWithTrace(ctx, rst.logger)

	// Add attributes to the span
	span.SetAttributes(attribute.String("redis.key", key))

	// Log the operation
	log.Info().Str("key", key).Msg("Getting value from Redis")

	result, err := rst.client.Get(ctx, key).Result()
	if err != nil {
		span.RecordError(err)
		log.Error().Err(err).Str("key", key).Msg("Failed to get value from Redis")

		return "", errors.Wrap(err, "failed to get value")
	}

	log.Info().Str("key", key).Msg("Successfully retrieved value from Redis")

	return result, nil
}

func (rst *RStorage) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	// Start a span for the Set operation
	ctx, span := rst.tracer.Start(ctx, "Redis.Set")
	defer span.End()

	log := logger.GetWithTrace(ctx, rst.logger)

	// Add attributes to the span
	span.SetAttributes(
		attribute.String("redis.key", key),
		attribute.String("redis.value", value), // Be cautious about logging sensitive values
		attribute.Int64("redis.expiration_ms", expiration.Milliseconds()),
	)

	// Log the operation
	log.Info().
		Str("key", key).
		Dur("expiration", expiration).
		Msg("Setting value in Redis")

	err := rst.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		span.RecordError(err)
		log.Error().Err(err).Str("key", key).Msg("Failed to set value in Redis")

		return errors.Wrap(err, "failed to set value")
	}

	log.Info().Str("key", key).Msg("Successfully set value in Redis")

	return nil
}
