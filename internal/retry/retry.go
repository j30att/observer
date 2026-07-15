package retry

import (
	"context"
	"time"
)

// DefaultDelays are the retry waits used by network and database operations.
var DefaultDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

// Do runs operation and retries it after each delay while errors are retriable.
func Do(ctx context.Context, delays []time.Duration, isRetriable func(error) bool, operation func() error) error {
	var err error
	for attempt := 0; attempt <= len(delays); attempt++ {
		err = operation()
		if err == nil {
			return nil
		}
		if !isRetriable(err) || attempt == len(delays) {
			return err
		}

		timer := time.NewTimer(delays[attempt])
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return err
}
