package fetcher_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"goscrapy/internal/fetcher"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type closeAwareBody struct {
	reader *strings.Reader
	closed bool
}

func (b *closeAwareBody) Read(p []byte) (int, error) {
	if b.closed {
		return 0, errors.New("response body is closed")
	}
	return b.reader.Read(p)
}

func (b *closeAwareBody) Close() error {
	b.closed = true
	return nil
}

func TestRobotsCacheAppliesRulesFromResponseBody(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := &closeAwareBody{reader: strings.NewReader("User-agent: *\nDisallow: /private\n")}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
			Request:    req,
		}, nil
	})
	cache := fetcher.NewRobotsCache(&http.Client{Transport: transport}, fetcher.DefaultUA)

	allowed, delay, err := cache.Allowed(context.Background(), "https://partner.example/private/report")
	if err != nil {
		t.Fatalf("robots check failed: %v", err)
	}
	if allowed {
		t.Fatal("path denied by robots.txt was allowed")
	}
	if delay != 0 {
		t.Fatalf("unexpected crawl delay: %s", delay)
	}
}
