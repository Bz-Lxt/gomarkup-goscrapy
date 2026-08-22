package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"goscrapy/internal/ratelimit"
)

func TestLimiterRefillsAfterBurst(t *testing.T) {
	limiter := ratelimit.NewLimiter(5)
	for i := 0; i < 5; i++ {
		if err := limiter.Wait(context.Background(), "example.com", 5); err != nil {
			t.Fatalf("consume initial token %d: %v", i, err)
		}
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- limiter.Wait(context.Background(), "example.com", 5)
	}()

	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("wait for refilled token: %v", err)
		}
	case <-time.After(750 * time.Millisecond):
		t.Fatal("waiter did not receive a refilled token")
	}

	updateDone := make(chan struct{})
	go func() {
		limiter.SetQPS("example.com", 10)
		close(updateDone)
	}()
	select {
	case <-updateDone:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("rate update remained blocked after the waiter completed")
	}
}
