package fetcher

import (
	"context"
	"errors"
	"math"
	"time"
)

func Backoff(attempt int, base time.Duration) time.Duration {
	if base <= 0 {
		base = 200 * time.Millisecond
	}
	if attempt < 0 {
		attempt = 0
	}
	if attempt > 6 {
		attempt = 6
	}
	d := float64(base) * math.Pow(2, float64(attempt))
	if d > float64(8*time.Second) {
		d = float64(8 * time.Second)
	}
	return time.Duration(d)
}

func Retryable(status int, err error) bool {
	if err != nil {
		// Redirect loops or too-many-redirect failures will not resolve by
		// retrying; surface them so upstream can alert/ignore accordingly.
		if errors.Is(err, ErrTooManyRedirects) {
			return false
		}
		return true
	}
	return status == 429 || status == 502 || status == 503 || status == 504
}

func SleepBackoff(ctx context.Context, attempt int) error {
	t := time.NewTimer(Backoff(attempt, 200*time.Millisecond))
	select {
	case <-ctx.Done():
		t.Stop()
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
