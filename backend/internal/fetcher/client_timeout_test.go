package fetcher

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestFetchPreservesClientTimeoutCause(t *testing.T) {
	client := New(nil, 20*time.Millisecond)
	client.base.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	_, err := client.Fetch(context.Background(), "http://slow-origin.test/page", false)
	if err == nil {
		t.Fatal("Fetch returned nil error for a response-header timeout")
	}
	if !strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
		t.Fatalf("Fetch error = %q, want client timeout detail", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("errors.Is(%q, context.DeadlineExceeded) = false, want true", err)
	}
}
