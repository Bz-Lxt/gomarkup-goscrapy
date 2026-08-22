package fetcher

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchAndRobots(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\nDisallow: /secret\n"))
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != DefaultUA {
			t.Errorf("ua=%s", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte("<html><body>hi</body></html>"))
	})
	mux.HandleFunc("/secret", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("nope"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(nil, 3*time.Second)
	ctx := context.Background()
	res, err := c.Fetch(ctx, srv.URL+"/ok", true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != 200 || !strings.Contains(string(res.Body), "hi") {
		t.Fatalf("fetch %+v", res)
	}
	blocked, err := c.Fetch(ctx, srv.URL+"/secret", true)
	if err != nil {
		t.Fatal(err)
	}
	if !blocked.RobotsSkip {
		t.Fatal("expected robots skip")
	}
}

func TestBackoff(t *testing.T) {
	if Backoff(0, time.Second) != time.Second {
		t.Fatal("base")
	}
	if Backoff(3, 200*time.Millisecond) < time.Second {
		t.Fatal("exp")
	}
	if !Retryable(429, nil) || Retryable(200, nil) {
		t.Fatal("retryable")
	}
}

func TestFetchCancelDuringRedirect(t *testing.T) {
	redirected := make(chan struct{})
	released := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/slow", http.StatusFound)
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		close(redirected)
		select {
		case <-r.Context().Done():
		case <-released:
		}
	})
	srv := httptest.NewServer(mux)
	defer func() {
		close(released)
		srv.Close()
	}()

	client := New(nil, 2*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.Fetch(ctx, srv.URL+"/start", false)
		done <- err
	}()

	select {
	case <-redirected:
	case <-time.After(time.Second):
		t.Fatal("redirect target was not requested")
	}
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Fetch error = %v, want context.Canceled", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Fetch did not stop after its context was canceled")
	}
}
