package fetcher

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestRobotsAllowedCancel verifies that when the parent context is cancelled
// while the robots.txt request is in-flight (slow server), Allowed returns
// context.Canceled promptly instead of blocking until the HTTP client timeout.
func TestRobotsAllowedCancel(t *testing.T) {
	// Slow robots server: holds the request open until it is cancelled.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the request's context is cancelled.
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cache := NewRobotsCache(http.DefaultClient, DefaultUA)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay to simulate task cancellation while
	// the worker is blocked in the first robots.txt check.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, _, err := cache.Allowed(ctx, srv.URL+"/page")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from Allowed when context cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	// Should return near-immediately after cancellation, not after the
	// 8s HTTP timeout.
	if elapsed > 2*time.Second {
		t.Fatalf("Allowed did not abort promptly, elapsed=%s", elapsed)
	}
}

// TestRobotsAllowedCacheHit confirms a cached rule is returned without
// re-fetching.
func TestRobotsAllowedCacheHit(t *testing.T) {
	var hits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
	})
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cache := NewRobotsCache(http.DefaultClient, DefaultUA)
	ctx := context.Background()

	if _, _, err := cache.Allowed(ctx, srv.URL+"/page"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cache.Allowed(ctx, srv.URL+"/page2"); err != nil {
		t.Fatal(err)
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("expected 1 robots fetch, got %d", got)
	}
}
