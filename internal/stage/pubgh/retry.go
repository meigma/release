package pubgh

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	// retryAttempts is one initial call plus three octokit retries.
	retryAttempts = 4
	// retryWait is the first backoff; later waits double (1s, 2s, 4s).
	retryWait = time.Second
)

// retryOp runs op up to [retryAttempts] times when the error is [ErrRetryable].
//
// A cancelled context and a failed sleep return unwrapped so callers can
// classify them. Non-retryable errors from op are wrapped as "step: err".
// Exhausted retryable failures wrap the last error as "step after N attempts".
func retryOp(ctx context.Context, sleep SleepFunc, step string, op func() error) error {
	var lastErr error
	for attempt := 1; attempt <= retryAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := op()
		if err == nil {
			return nil
		}
		lastErr = err
		if !errors.Is(err, ErrRetryable) || attempt == retryAttempts {
			if errors.Is(err, ErrRetryable) {
				return fmt.Errorf("%s after %d attempts: %w", step, attempt, err)
			}

			return fmt.Errorf("%s: %w", step, err)
		}
		if err := sleep(ctx, retryWait<<(attempt-1)); err != nil {
			return err
		}
	}
	if lastErr == nil {
		return errors.New("retry budget exhausted")
	}

	return fmt.Errorf("%s after %d attempts: %w", step, retryAttempts, lastErr)
}

// sleepContext waits for d or returns ctx.Err() if the context ends first.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
