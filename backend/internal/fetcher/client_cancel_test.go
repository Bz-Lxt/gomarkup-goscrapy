package fetcher

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type blockingRobotsTransport struct {
	started chan struct{}
	release chan struct{}
}

func (t *blockingRobotsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	close(t.started)
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-t.release:
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("User-agent: *\nAllow: /\n")),
			Request:    req,
		}, nil
	}
}

func TestFetchCancellationStopsRobotsRequest(t *testing.T) {
	robotsStarted := make(chan struct{})
	releaseRobots := make(chan struct{})

	client := New(nil, 5*time.Second)
	client.robots.client = &http.Client{Transport: &blockingRobotsTransport{
		started: robotsStarted,
		release: releaseRobots,
	}}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := client.Fetch(ctx, "http://shop.example/products/42", true)
		result <- err
	}()

	select {
	case <-robotsStarted:
	case <-time.After(time.Second):
		close(releaseRobots)
		t.Fatal("robots request did not start")
	}
	cancel()

	select {
	case err := <-result:
		close(releaseRobots)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Fetch error = %v, want context.Canceled", err)
		}
	case <-time.After(500 * time.Millisecond):
		close(releaseRobots)
		<-result
		t.Fatal("Fetch did not return promptly after cancellation")
	}
}
