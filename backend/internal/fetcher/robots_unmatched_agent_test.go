package fetcher

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRobotsAllowsPageWhenNoUserAgentMatches(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch r.URL.Path {
		case "/robots.txt":
			body = "User-agent: OtherBot\nDisallow: /private\n"
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("not found")), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})

	robots := NewRobotsCache(&http.Client{Transport: transport}, DefaultUA)
	allowed, delay, err := robots.Allowed(context.Background(), "http://example.test/products")
	if err != nil {
		t.Fatalf("Allowed() error = %v", err)
	}
	if !allowed {
		t.Fatal("Allowed() = false, want true when no user-agent group matches")
	}
	if delay != 0 {
		t.Fatalf("Allowed() delay = %v, want 0", delay)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
