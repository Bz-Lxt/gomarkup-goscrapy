package renderer_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"goscrapy/internal/fetcher"
	"goscrapy/internal/renderer"
)

func TestCaptureHonorsCanceledContextDuringStaticFallback(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte("<html><body>late response</body></html>"))
	}))
	defer srv.Close()

	svc := renderer.NewService("", fetcher.New(nil, 200*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	started := time.Now()
	_, err := svc.Capture(ctx, srv.URL)
	elapsed := time.Since(started)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Capture error = %v, want context.Canceled", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("Capture returned after %v, want prompt cancellation", elapsed)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("target received %d requests after cancellation, want 0", got)
	}
}
