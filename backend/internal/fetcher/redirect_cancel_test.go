package fetcher

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestRedirectFollowsContextCancel verifies that cancelling the task context
// aborts an in-flight redirect chain promptly, instead of waiting for the
// HTTP client timeout.
func TestRedirectFollowsContextCancel(t *testing.T) {
	// target handler — blocks forever until cancelled; we just need it
	// to never return on its own so the only way out is context cancel.
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
	}))
	defer target.Close()

	// redirect server: always 302 -> target
	var hops int32
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hops, 1)
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirect.Close()

	c := New(nil, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := c.Fetch(ctx, redirect.URL, false)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled fetch, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// Should return well before the 10s timeout.
	if elapsed > 5*time.Second {
		t.Fatalf("fetch did not follow context cancel promptly: %v", elapsed)
	}
	t.Logf("returned after %v, hops=%d", elapsed, atomic.LoadInt32(&hops))
}

// TestRedirectFollowsContextCancel_Direct ensures non-redirect pages still
// cancel normally (baseline / regression guard).
func TestRedirectFollowsContextCancel_Direct(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
	}))
	defer target.Close()

	c := New(nil, 10*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := c.Fetch(ctx, target.URL, false)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled fetch, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("fetch did not follow context cancel promptly: %v", elapsed)
	}
}

// TestRedirectFollowsContextCancel_MultiHop verifies cancellation works
// across a chain of several 302 hops.
func TestRedirectFollowsContextCancel_MultiHop(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
	}))
	defer target.Close()

	// Build a chain: hop0 -> hop1 -> hop2 -> ... -> hop3 -> target
	// Each hop redirects to the next; the final one redirects to target.
	servers := make([]*httptest.Server, 4)
	for i := 0; i < len(servers); i++ {
		i := i
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if i == len(servers)-1 {
				http.Redirect(w, r, target.URL, http.StatusFound)
				return
			}
			http.Redirect(w, r, servers[i+1].URL, http.StatusFound)
		}))
	}
	defer func() {
		for _, s := range servers {
			s.Close()
		}
	}()

	c := New(nil, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := c.Fetch(ctx, servers[0].URL, false)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled fetch, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("fetch did not follow context cancel promptly: %v", elapsed)
	}
}

// TestRedirectSuccess ensures normal redirects still complete and the
// FinalURL reflects the last hop.
func TestRedirectSuccess(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><body>hello</body></html>")
	}))
	defer final.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redirect.Close()

	c := New(nil, 5*time.Second)
	res, err := c.Fetch(context.Background(), redirect.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != 200 {
		t.Fatalf("expected 200, got %d", res.Status)
	}
	if res.FinalURL != final.URL {
		t.Fatalf("expected final URL %s, got %s", final.URL, res.FinalURL)
	}
}
