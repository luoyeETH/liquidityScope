package indexer

import (
	"context"
	"time"
)

func withRetry(ctx context.Context, maxRetries int, baseDelay time.Duration, fn func(context.Context) error) error {
	if maxRetries < 0 {
		maxRetries = 0
	}
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond
	}

	delay := baseDelay
	for attempt := 0; ; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if attempt >= maxRetries {
			return err
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		delay *= 2
	}
}
