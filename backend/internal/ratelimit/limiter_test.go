package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucketSpacing(t *testing.T) {
	l := NewLimiter(50)
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := l.Wait(ctx, "example.test", 50); err != nil {
			t.Fatal(err)
		}
	}
	if time.Since(start) > time.Second {
		t.Fatal("unexpectedly slow")
	}
}

func TestAdaptiveBackoff(t *testing.T) {
	l := NewLimiter(4)
	a := NewAdaptive(l)
	q := 4.0
	for i := 0; i < 8; i++ {
		q = a.Observe("x.test", 429, 50*time.Millisecond, 4)
	}
	if q >= 4 {
		t.Fatalf("expected qps drop, got %v", q)
	}
	a.ApplyCommand("x.test", 1.5)
	if a.QPS("x.test") != 1.5 {
		t.Fatalf("apply %v", a.QPS("x.test"))
	}
}
