package renderer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"goscrapy/internal/fetcher"
)

// slowHandler sleeps for a while before responding, and counts how many
// times it was hit.
func slowHandler(delay time.Duration, hitCount *int32) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hitCount, 1)
		select {
		case <-r.Context().Done():
		case <-time.After(delay):
		}
		_, _ = w.Write([]byte("<html><body>hi</body></html>"))
	}
}

// TestCaptureStaticRespectsContextCancel verifies that when the caller's
// context is cancelled, CaptureStatic stops promptly and returns
// context.Canceled instead of waiting for the fetch timeout.
func TestCaptureStaticRespectsContextCancel(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(slowHandler(30*time.Second, &hits)))
	defer srv.Close()

	// Use a long fetcher timeout so the only thing that stops the request is
	// context cancellation, not the http client timeout.
	client := fetcher.New(nil, 60*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := CaptureStatic(ctx, client, srv.URL)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	// Should return quickly, not after the server's 30s delay.
	if elapsed > 5*time.Second {
		t.Fatalf("CaptureStatic took too long to cancel: %v", elapsed)
	}
}

// TestServiceCaptureCancelBeforeFetch verifies that an already-cancelled
// context causes Capture to return immediately without hitting the network.
func TestServiceCaptureCancelBeforeFetch(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(slowHandler(30*time.Second, &hits)))
	defer srv.Close()

	client := fetcher.New(nil, 60*time.Second)
	svc := NewService("", client) // no CDP ws, static-only path

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	_, err := svc.Capture(ctx, srv.URL)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("expected 0 network hits, got %d", atomic.LoadInt32(&hits))
	}
}

// TestServiceCaptureStaticFallbackCancel verifies that when CDP is not
// configured, the static fallback uses the caller's context and stops on
// cancel.
func TestServiceCaptureStaticFallbackCancel(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(slowHandler(30*time.Second, &hits)))
	defer srv.Close()

	client := fetcher.New(nil, 60*time.Second)
	svc := NewService("", client)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := svc.Capture(ctx, srv.URL)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("Capture took too long to cancel: %v", elapsed)
	}
}

// TestServiceCaptureNoCDPStaticSuccess verifies the happy path still works.
func TestServiceCaptureNoCDPStaticSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("<html><body>hello</body></html>"))
	}))
	defer srv.Close()

	client := fetcher.New(nil, 10*time.Second)
	svc := NewService("", client)

	rec, err := svc.Capture(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec == nil || rec.Source != "static" {
		t.Fatalf("unexpected record: %+v", rec)
	}
	if rec.HTML == "" {
		t.Fatal("empty html")
	}
}

// TestCaptureStaticDeadlineExceeded verifies context.DeadlineExceeded is
// also propagated (not swallowed as a generic error).
func TestCaptureStaticDeadlineExceeded(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(slowHandler(30*time.Second, &hits)))
	defer srv.Close()

	client := fetcher.New(nil, 60*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := CaptureStatic(ctx, client, srv.URL)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func ExampleService_Capture() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><body>ok</body></html>")
	}))
	defer srv.Close()

	svc := NewService("", fetcher.New(nil, 5*time.Second))
	rec, err := svc.Capture(context.Background(), srv.URL)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("source:", rec.Source)
	// Output: source: static
}
