package utils

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
)

const maxRetries = 5

type OperationFunc func() error

// RetryOperation executes a database operation with retry logic.
func RetryOperation(ctx context.Context, operation OperationFunc) error {
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = 1 * time.Second
	backoffConfig.Multiplier = 3
	retryWithLimit := backoff.WithMaxRetries(backoffConfig, maxRetries)

	err := backoff.Retry(func() error {
		err := operation()
		if err != nil {
			return err
		}

		return backoff.Permanent(err)
	}, backoff.WithContext(retryWithLimit, ctx))

	return errors.Wrap(err, "failed to execute operation after retry")
}
