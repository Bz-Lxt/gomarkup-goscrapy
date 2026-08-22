package fetcher_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"goscrapy/internal/fetcher"
)

type closeTrackingBody struct {
	io.Reader
	closed atomic.Bool
}

func (b *closeTrackingBody) Close() error {
	b.closed.Store(true)
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRobotsCacheClosesErrorResponseBody(t *testing.T) {
	body := &closeTrackingBody{Reader: strings.NewReader("not found")}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       body,
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}

	cache := fetcher.NewRobotsCache(client, "crawler-test")
	allowed, _, err := cache.Allowed(context.Background(), "https://example.test/products/42")
	if err != nil {
		t.Fatalf("Allowed returned an error for a missing robots.txt: %v", err)
	}
	if !allowed {
		t.Fatal("missing robots.txt unexpectedly blocked the page")
	}
	if !body.closed.Load() {
		t.Fatal("robots.txt response body remained open after Allowed returned")
	}
}
